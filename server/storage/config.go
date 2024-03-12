package storage

type Config struct {
	DatabaseUrl string
}

func NewConfig() *Config {
	return &Config{
		DatabaseUrl: "host=db dbname=RestApiServer sslmode=disable user=chebryakov password=password",
	}
}
