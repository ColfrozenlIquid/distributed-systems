package main

import (
	"http/internal/request"
	"http/internal/response"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const port = 8080

func main() {
	server, err := Serve(port, func(w *response.Writer, req *request.Request) {

		body := []byte("Hello, world!")
		headers := response.GetDefaultHeaders(len(body))
		w.WriteStatusLine(response.OK)
		w.WriteHeaders(headers)
		w.WriteBody(body)

	})
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
