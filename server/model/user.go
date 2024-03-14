package model

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID              int       `json:"id"`
	UserEmail       string    `json:"email"`
	UserName        string    `json:"name"`
	UserPassword    string    `json:"-"`
	UserSurname     string    `json:"surname"`
	UserPhoneNumber string    `json:"phonenumber"`
	UserBirthday    time.Time `json:"birthday"`
	UserLogin       string    `json:"login"`
}

func CreateUserTable(db *sql.DB) error {
	query := `DROP TABLE IF EXISTS users`
	_, err := db.Exec(query)
	query = `CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL NOT NULL PRIMARY KEY,
		user_email VARCHAR(255) UNIQUE,
		user_name VARCHAR(100),
		user_login VARCHAR(100) NOT NULL UNIQUE,
		user_surname VARCHAR(100),
		user_phone_number VARCHAR(12) UNIQUE,
		user_password VARCHAR(255) NOT NULL,
		user_birthday DATE
	)`

	_, err = db.Exec(query)

	if err != nil {
		return err
	}

	return nil

}

func (u *User) IsPasswordEqual(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.UserPassword), []byte(password)) == nil
}
