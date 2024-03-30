package storage

import (
	"database/sql"
	"main_service/model"

	_ "github.com/lib/pq"
)

type Storage struct {
	Config         *Config
	DB             *sql.DB
	UserRepository *UserRepository
}

func New(config *Config) *Storage {
	return &Storage{
		Config: config,
	}
}

func (storage *Storage) ConnnectToStorage() error {
	db, err := sql.Open("postgres", storage.Config.DatabaseUrl)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	storage.DB = db

	err = model.CreateUserTable(storage.DB)

	if err != nil {
		return err
	}

	return nil

}

func (s *Storage) CloseConnection() {
	return
}

func (storage *Storage) User() *UserRepository {
	if storage.UserRepository != nil {
		return storage.UserRepository
	}

	storage.UserRepository = &UserRepository{
		storage: storage,
	}
	return storage.UserRepository
}
