package main

import (
	"control-center/internal/client"
	"control-center/internal/config"
	"control-center/internal/model"
	"control-center/internal/routes"
	"log"
	"os"
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
	defer client.WorldClient.Close()

	state := model.NewState(client.WorldClient.WorldServiceClient)

	router := vshttp.NewRouter()
	routes.Routes(router, state)

	server, err := vshttp.Serve(":8080", router)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on :8080")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
