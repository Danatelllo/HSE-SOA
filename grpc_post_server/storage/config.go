package storage

type StorageConfig struct {
	DatabaseUrl string
}

func NewConfig() *StorageConfig {
	return &StorageConfig{
		DatabaseUrl: "host=post_db dbname=GrpcPostServer sslmode=disable user=chebryakov password=password",
	}
}
