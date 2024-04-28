package storage

type StorageConfig struct {
	DatabaseUrl string
}

func NewConfig() *StorageConfig {
	return &StorageConfig{
		DatabaseUrl: "host=db dbname=MetricDB sslmode=disable user=chebryakov password=password",
	}
}
