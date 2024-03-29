package storage

import (
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	Login string
	Title string
	Body  string
	Date  time.Time
}

func ConnnectToStorage(Config *StorageConfig) (*gorm.DB, error) {
	dbMaster, err := gorm.Open(postgres.Open(Config.DatabaseUrl), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil, err
	}

	dbMaster.AutoMigrate(&Post{})

	return dbMaster, nil
}

func New(Config *StorageConfig) (*gorm.DB, error) {
	return ConnnectToStorage(Config)
}
