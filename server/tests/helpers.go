package tests

import (
	"fmt"
	"main_service/storage"
	"strings"
	"testing"
)

func TestStore(t *testing.T, databaseURL string) (*storage.Storage, func(...string)) {
	t.Helper()
	config := storage.NewConfig()
	config.DatabaseUrl = databaseURL
	storage := storage.New(config)
	if err := storage.ConnnectToStorage(); err != nil {
		t.Fatal(err)
	}

	return storage, func(tableNames ...string) {
		if len(tableNames) > 0 {
			if _, err := storage.DB.Exec(fmt.Sprintf("TRUNCATE %s CASCADE", strings.Join(tableNames, ", "))); err != nil {
				t.Fatal()
			}
		}
		storage.CloseConnection()
	}
}
