package service

import "grpc_post_server/storage"

type GrpcPostServerConfig struct {
	Port          string
	StorageConfig *storage.StorageConfig
}

func NewRestApiConfig() *GrpcPostServerConfig {
	return &GrpcPostServerConfig{
		Port:          ":8081",
		StorageConfig: storage.NewConfig(),
	}
}
