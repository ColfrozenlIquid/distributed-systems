package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/client"
)

// One-shot job: creates a toxiproxy proxy per node (node-0..node-N)
// pointed at the real broker, then applies ONE latency+jitter value
// to every node's proxy (both directions). Run once after toxiproxy
// is healthy and before the node containers start (nodes retry-connect,
// so exact ordering isn't critical).

const toxiproxyAPIPort = "8474"

type Scenario struct {
	Latency time.Duration
	Jitter  time.Duration
}

var scenarios = map[string]Scenario{
	"uniform": {Latency: 20 * time.Millisecond, Jitter: 5 * time.Millisecond},
	"fast":    {Latency: 2 * time.Millisecond, Jitter: 1 * time.Millisecond},
	"chaos":   {Latency: 100 * time.Millisecond, Jitter: 50 * time.Millisecond},
}

func main() {
	numNodes := 20
	if v := os.Getenv("NUM_NODES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			numNodes = n
		}
	}
	brokerAddr := envOr("BROKER_ADDR", "mqtt-broker:1883")
	toxiproxyHost := envOr("TOXIPROXY_HOST", "toxiproxy")
	basePort := 21000

	nodeIDs := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		nodeIDs[i] = fmt.Sprintf("node-%d", i)
	}

	client := toxiproxy.NewClient(fmt.Sprintf("%s:%s", toxiproxyHost, toxiproxyAPIPort))

	var proxies map[string]*toxiproxy.Proxy
	for attempt := 0; attempt < 20; attempt++ {
		p, err := setupProxies(client, nodeIDs, brokerAddr, basePort)
		if err == nil {
			proxies = p
			break
		}
		log.Printf("toxiproxy not ready yet (attempt %d): %v", attempt, err)
		time.Sleep(1 * time.Second)
	}
	if proxies == nil {
		log.Fatal("could not create proxies: toxiproxy never became ready")
	}

	scenarioName := envOr("SCENARIO", "uniform")
	s, ok := scenarios[scenarioName]
	if !ok {
		log.Fatalf("unknown scenario %q (options: %v)", scenarioName, scenarioNames())
	}

	applyScenario(proxies, s, nodeIDs)
	log.Printf("proxy-setup complete: %d proxies configured, scenario=%s (latency=%s jitter=%s)",
		len(proxies), scenarioName, s.Latency, s.Jitter)
}

func scenarioNames() []string {
	names := make([]string, 0, len(scenarios))
	for k := range scenarios {
		names = append(names, k)
	}
	return names
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func setupProxies(client *toxiproxy.Client, nodes []string, brokerAddr string, basePort int) (map[string]*toxiproxy.Proxy, error) {
	proxies := make(map[string]*toxiproxy.Proxy)
	for i, id := range nodes {
		listenPort := basePort + i
		p, err := client.CreateProxy(id, fmt.Sprintf("0.0.0.0:%d", listenPort), brokerAddr)
		if err != nil {
			return nil, fmt.Errorf("create proxy for %s: %w", id, err)
		}
		proxies[id] = p
	}
	return proxies, nil
}

// applyScenario adds the same latency toxic (both directions) to
// every node's proxy. Nothing per-node, nothing per-pair.
func applyScenario(proxies map[string]*toxiproxy.Proxy, s Scenario, nodes []string) {
	attrs := toxiproxy.Attributes{
		"latency": s.Latency.Milliseconds(),
		"jitter":  s.Jitter.Milliseconds(),
	}

	for _, id := range nodes {
		p, ok := proxies[id]
		if !ok {
			continue
		}
		if _, err := p.AddToxic("latency_upstream", "latency", "upstream", 1.0, attrs); err != nil {
			log.Printf("failed to add upstream toxic for %s: %v", id, err)
		}
		if _, err := p.AddToxic("latency_downstream", "latency", "downstream", 1.0, attrs); err != nil {
			log.Printf("failed to add downstream toxic for %s: %v", id, err)
		}
	}
}
