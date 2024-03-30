package service

import "main_service/storage"

type RestApiConfig struct {
	Port          string
	Host          string
	StorageConfig *storage.Config
}

func NewRestApiConfig() *RestApiConfig {
	return &RestApiConfig{
		Port:          ":8081",
		Host:          "0.0.0.0",
		StorageConfig: storage.NewConfig(),
	}
}
