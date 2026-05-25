package main

import (
	"actors/internal/actor"
	"actors/internal/client"
	"actors/internal/config"
	"context"
	"fmt"
	"log"
	"net/http"
	"shared"
	"time"

	"go.uber.org/zap"
)

type App struct {
	logger *zap.Logger
	client client.Client
	actor  actor.Actor
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	actorCfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("failed to read sensor config", zap.Error(err))
	}

	clientCfg := client.ClientConfig{
		HttpAddress: actorCfg.BaseStationEndpoint,
		GrpcAddress: actorCfg.WorldSimulationEndpoint,
	}

	client, err := client.NewClient(clientCfg, logger)
	defer client.Grpc.Close()

	actor, err := actor.NewActor(*actorCfg, client.Grpc.Grpc)
	if err != nil {
		logger.Fatal("failed to create new actor", zap.Error(err))
	}

	app := App{
		logger: logger,
		client: *client,
		actor:  actor,
	}

	err = app.Register()
	if err != nil {
		logger.Fatal("failed to register actor")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Run(ctx, actor, clientCfg.HttpAddress)
}

func (s *App) Register() error {
	payload := shared.ActorRegistration{
		Actor: shared.ActorInfo{
			Id: s.actor.Id(),
			Position: shared.Position{
				X: s.actor.Position().X,
				Y: s.actor.Position().Y,
			},
			ActorType: shared.ActorType(s.actor.Type()),
		},
	}

	resp, err := s.client.Http.Post("/register/actor", payload)
	if err != nil {
		return fmt.Errorf("post register actor: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		s.logger.Error("registration failed", zap.Any("response", resp))
	}

	return nil
}

func (s *App) Run(ctx context.Context, actor actor.Actor, endpoint string) {
	ticker := time.NewTicker(actor.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.logger.Info("actor should do something here")
		}
	}
}
