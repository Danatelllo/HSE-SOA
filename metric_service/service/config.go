package service

import "metric_service/storage"

type MetricServerConfig struct {
	Host          string
	Port          string
	StorageConfig *storage.StorageConfig
	KafkaConsumer string
}

func NewMetricServerConfig() *MetricServerConfig {
	return &MetricServerConfig{
		Host:          "0.0.0.0",
		Port:          ":5432",
		StorageConfig: storage.NewConfig(),
		KafkaConsumer: "kafka:29092",
	}
}
