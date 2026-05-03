package main

import (
	"fmt"
	"http/internal/request"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		req, err := request.FromReader(conn)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Request line: \n")
		fmt.Printf("- Method: %s\n", req.RequestLine.Method)
		fmt.Printf("- Target: %s\n", req.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", req.RequestLine.HttpVersion)
		req.Headers.ForEach(func(key, value string) {
			fmt.Printf("- %s: %s\n", key, value)
		})
		fmt.Printf("Body:\n")
		fmt.Printf("%s\n", req.Body)
	}
}
