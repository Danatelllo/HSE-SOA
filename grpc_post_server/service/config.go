package service

import "grpc_post_server/storage"

type GrpcTaskServerConfig struct {
	Port          string
	StorageConfig *storage.StorageConfig
}

func NewGrpcTaskServerConfig() *GrpcTaskServerConfig {
	return &GrpcTaskServerConfig{
		Port:          "0.0.0.0:8080",
		StorageConfig: storage.NewConfig(),
	}
}
