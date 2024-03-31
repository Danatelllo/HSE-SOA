package main

import (
	"grpc_post_server/service"
	"log"
)

func main() {
	config := service.NewGrpcTaskServerConfig()
	s := service.NewGrpcTaskServer(config)
	if error := s.Start(); error != nil {
		log.Fatal(error)
	}
}
