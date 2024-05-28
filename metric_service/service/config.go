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
		Port:          "0.0.0.0:9000",
		StorageConfig: storage.NewConfig(),
		KafkaConsumer: "kafka:29092",
	}
}
