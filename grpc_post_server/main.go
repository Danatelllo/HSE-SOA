package main

import (
	"grpc_post_server/service"
	"log"
)

func main() {
	config := service.NewRestApiConfig()
	s := service.NewRestApiServer(config)
	if error := s.Start(); error != nil {
		log.Fatal(error)
	}
}
