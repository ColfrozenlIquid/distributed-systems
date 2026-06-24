package main

import (
	"context"
	"control-center/internal/client"
	"control-center/internal/config"
	"control-center/internal/model"
	"control-center/internal/routes"
	"log"
	"os/signal"
	"syscall"
	"vshttp"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("failed to read control-center config", zap.Error(err))
	}
	client, err := client.NewClient(*cfg, logger)
	if err != nil {
		logger.Fatal("failed to create control-center clients", zap.Error(err))
	}
	defer client.WorldClient.Close()
	defer client.MqttClient.Disconnect(250)

	state := model.NewState(client.WorldClient.WorldServiceClient)

	router := vshttp.NewRouter()
	routes.Routes(router, state, client.ActorClient)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go state.StartReactiveMessageProcessing(ctx, client.MqttClient)

	server, err := vshttp.Serve(":8080", router)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()

	log.Println("Server started on :8080")
	<-ctx.Done()
	log.Println("Server gracefully stopped")
}
