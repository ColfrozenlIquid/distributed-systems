package main

import (
	"actors/internal/actor"
	"actors/internal/client"
	cl "actors/internal/client"
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
	client cl.Client
	actor  actor.Actor
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	logger.Info("Initializationn started")

	actorCfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("failed to read actor config", zap.Error(err))
	}

	clientCfg := client.ClientConfig{
		HttpAddress: actorCfg.BaseStationEndpoint,
		GrpcAddress: actorCfg.WorldSimulationEndpoint,
	}

	client, err := client.NewClient(clientCfg, logger)
	if err != nil {
		logger.Fatal("creating client failed", zap.Error(err))
	}
	defer client.Grpc.Close()

	actor, err := actor.NewActor(*actorCfg, client.Grpc.WorldServiceClient, logger)
	if err != nil {
		logger.Fatal("failed to create new actor", zap.Error(err))
	}

	handler := cl.NewActorHandler(logger, actor, client.Grpc.WorldServiceClient)

	client.Grpc.RegisterActorHandler(handler)

	go func() {
		if err := client.Grpc.Serve(); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

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

	logger.Info("Initializationn successful")

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
			s.actor.Update(ctx)
		}
	}
}
