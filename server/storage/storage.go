package storage

import (
	"server/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage struct {
	Config         *Config
	DB             *gorm.DB
	UserRepository *UserRepository
}

func New(config *Config) *Storage {
	return &Storage{
		Config: config,
	}
}

func (storage *Storage) ConnectToStorage() error {
	db, err := gorm.Open(postgres.Open(storage.Config.DatabaseUrl), &gorm.Config{})
	if err != nil {
		return err
	}

	if !db.Migrator().HasTable(&model.User{}) {
		err = db.Debug().Migrator().CreateTable(&model.User{})
		if err != nil {
			return err
		}
	}

	storage.DB = db
	return nil
}

func (storage *Storage) CloseConnection() error {
	sqlDB, err := storage.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (storage *Storage) User() *UserRepository {
	if storage.UserRepository != nil {
		return storage.UserRepository
	}

	storage.UserRepository = &UserRepository{
		DB: storage.DB,
	}
	return storage.UserRepository
}
