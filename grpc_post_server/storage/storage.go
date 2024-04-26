package storage

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	Login string
	Title string
	Body  string
	Date  time.Time
}

func ConnnectToStorage(Config *StorageConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(Config.DatabaseUrl), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if !db.Migrator().HasTable(&Task{}) {
		err = db.Debug().Migrator().CreateTable(Task{})
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func New(Config *StorageConfig) (*gorm.DB, error) {
	return ConnnectToStorage(Config)
}
