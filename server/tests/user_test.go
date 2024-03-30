package tests

import (
	"main_service/model"
	"main_service/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserCreate(t *testing.T) {
	storage, clearTables := TestStore(t, "host=localhost dbname=RestApiServer_test sslmode=disable")
	defer clearTables("users")

	user, err := storage.User().Create(&model.User{
		UserEmail:       "tmp@example.com",
		UserPassword:    "password",
		UserLogin:       "Name",
		UserSurname:     "Surname",
		UserPhoneNumber: "+79957482342",
	})

	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func TestUserFindByLogin(t *testing.T) {
	storage, clearTables := TestStore(t, "host=localhost dbname=RestApiServer_test sslmode=disable")
	defer clearTables("users")
	login := "login"

	storage.User().Create(&model.User{
		UserEmail:    "tmp@example.com",
		UserPassword: "password",
		UserLogin:    login,
	})

	user, err := storage.User().FindUserByLogin(login)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, user.UserLogin, login)
}

func TestEncryption(t *testing.T) {
	password := "password"
	encryptPassword, err := storage.EncryptPassword(password)
	if err != nil {
		t.Fatal()
	}

	assert.NotEqual(t, password, encryptPassword)
	assert.NotEmpty(t, encryptPassword)
}
