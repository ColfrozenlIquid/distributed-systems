package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"vshttp"
)

func main() {
	router := vshttp.NewRouter()
	Routes(router)

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
