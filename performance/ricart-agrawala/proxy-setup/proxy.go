package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/client"
)

// This is a ONE-SHOT job: it creates a toxiproxy proxy per node
// (node-0..node-N) pointed at the real broker, applies a latency
// scenario, then exits. Run it once after toxiproxy is healthy and
// before the node containers start their control loop (a short
// sleep/healthcheck-based ordering in compose is enough — nodes
// will just retry-connect via SetAutoReconnect until the proxy exists).

const toxiproxyAPIPort = "8474" // NOTE: toxiproxy's real default API port is 8474, not 8478

type Matrix struct {
	base   map[string]map[string]time.Duration
	jitter map[string]map[string]time.Duration
}

func NewMatrix() *Matrix {
	return &Matrix{
		base:   make(map[string]map[string]time.Duration),
		jitter: make(map[string]map[string]time.Duration),
	}
}

func (m *Matrix) Set(from, to string, delay time.Duration) {
	if m.base[from] == nil {
		m.base[from] = make(map[string]time.Duration)
	}
	m.base[from][to] = delay
}

func (m *Matrix) SetJitter(from, to string, stddev time.Duration) {
	if m.jitter[from] == nil {
		m.jitter[from] = make(map[string]time.Duration)
	}
	m.jitter[from][to] = stddev
}

func Uniform(nodes []string, delay time.Duration) *Matrix {
	m := NewMatrix()
	for _, a := range nodes {
		for _, b := range nodes {
			if a != b {
				m.Set(a, b, delay)
			}
		}
	}
	return m
}

func SkewedSlowNode(nodes []string, baseDelay, slowDelay time.Duration, slowNode string) *Matrix {
	m := Uniform(nodes, baseDelay)
	for _, n := range nodes {
		if n == slowNode {
			continue
		}
		m.Set(slowNode, n, slowDelay)
		m.Set(n, slowNode, slowDelay)
	}
	return m
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

	// Wait for toxiproxy to be reachable (simple retry loop, since
	// compose depends_on/healthcheck only guarantees the container
	// process started, not that the API is accepting yet).
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

	scenario := envOr("SCENARIO", "uniform")
	var m *Matrix
	switch scenario {
	case "skewed":
		slowNode := envOr("SLOW_NODE", "node-3")
		m = SkewedSlowNode(nodeIDs, 20*time.Millisecond, 400*time.Millisecond, slowNode)
	default:
		m = Uniform(nodeIDs, 20*time.Millisecond)
	}
	// Optional uniform jitter on top of the base matrix
	for _, a := range nodeIDs {
		for _, b := range nodeIDs {
			if a != b {
				m.SetJitter(a, b, 5*time.Millisecond)
			}
		}
	}

	applyMatrixAsToxics(proxies, m, nodeIDs)
	log.Printf("proxy-setup complete: %d proxies configured, scenario=%s", len(proxies), scenario)
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

// applyMatrixAsToxics adds latency toxics on BOTH directions of each
// node's proxy link. "upstream" delays node->broker traffic (the
// node's own publishes, e.g. lock requests/replies); "downstream"
// delays broker->node traffic (messages arriving from other nodes).
// Both matter for synchronization delay, since the handoff involves
// a release message going out and other nodes' requests/replies
// coming back in.
//
// CAVEAT: toxics are per-proxy, i.e. per node's link to the shared
// broker — not truly per node-pair. We approximate each node's
// outbound/inbound toxic using the max delay to/from any peer. If
// pairwise asymmetry is essential to what you're measuring, this
// is a simplification worth revisiting (true pairwise control needs
// N^2 proxies).
func applyMatrixAsToxics(proxies map[string]*toxiproxy.Proxy, m *Matrix, nodes []string) {
	for _, from := range nodes {
		p, ok := proxies[from]
		if !ok {
			continue
		}
		var maxDelay, maxJitter time.Duration
		for _, to := range nodes {
			if to == from {
				continue
			}
			if d := m.base[from][to]; d > maxDelay {
				maxDelay = d
			}
			if j := m.jitter[from][to]; j > maxJitter {
				maxJitter = j
			}
		}

		attrs := toxiproxy.Attributes{
			"latency": maxDelay.Milliseconds(),
			"jitter":  maxJitter.Milliseconds(),
		}
		if _, err := p.AddToxic("latency_upstream", "latency", "upstream", 1.0, attrs); err != nil {
			log.Printf("failed to add upstream toxic for %s: %v", from, err)
		}
		if _, err := p.AddToxic("latency_downstream", "latency", "downstream", 1.0, attrs); err != nil {
			log.Printf("failed to add downstream toxic for %s: %v", from, err)
		}
	}
}
