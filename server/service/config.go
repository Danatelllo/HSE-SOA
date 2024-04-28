package service

import (
	"server/storage"
)

type RestApiConfig struct {
	Port              string
	Host              string
	TaskServerAddress string
	StorageConfig     *storage.Config
	KafkaConfig       []string
}

func NewRestApiConfig() *RestApiConfig {
	return &RestApiConfig{
		Port:              ":8081",
		Host:              "0.0.0.0",
		TaskServerAddress: "post_app:8080",
		StorageConfig:     storage.NewConfig(),
		KafkaConfig:       []string{"kafka:29092"},
	}
}
