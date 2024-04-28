package storage

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ViewInfo struct {
	gorm.Model
	TaskId    string
	UserLogin string
}

type LikeInfo struct {
	gorm.Model
	TaskId    string
	UserLogin string
}

func ConnnectToStorage(Config *StorageConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(Config.DatabaseUrl), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if !db.Migrator().HasTable(&LikeInfo{}) {
		err = db.Debug().Migrator().CreateTable(LikeInfo{})
		if err != nil {
			return nil, err
		}
	}

	if !db.Migrator().HasTable(&ViewInfo{}) {
		err = db.Debug().Migrator().CreateTable(ViewInfo{})
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func New(Config *StorageConfig) (*gorm.DB, error) {
	return ConnnectToStorage(Config)
}
