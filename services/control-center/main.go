package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"vshttp"
)

func main() {
	router := vshttp.NewRouter()
	routing(router)

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

func routing(router *vshttp.Router) {
	var name string

	router.Get("/name", func(c *vshttp.Context) error {
		name = c.Query("name")
		return c.NoContent()
	})

	router.Post("/name", func(c *vshttp.Context) error {
		return c.JSON(vshttp.OK, map[string]string{
			"message": fmt.Sprintf("Hello, %s!", name),
		})
	})

}
