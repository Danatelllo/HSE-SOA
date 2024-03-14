package main

import (
	"log"
	"main_service/service"
)

// @title Todo App API
// @version 1.0
// @description API Server for TodoList Application

// @host localhost:8081
// @BasePath /

func main() {
	config := service.NewRestApiConfig()
	s := service.NewRestApiServer(config)
	if error := s.Start(); error != nil {
		log.Fatal(error)
	}
}
