package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID              int       `json:"id"`
	UserEmail       string    `json:"email"`
	UserName        string    `json:"name"`
	UserPassword    string    `json:"-"`
	UserSurname     string    `json:"surname"`
	UserPhoneNumber string    `json:"phonenumber"`
	UserBirthday    time.Time `json:"birthday"`
	UserLogin       string    `json:"login"`
}

func (u *User) IsPasswordEqual(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.UserPassword), []byte(password)) == nil
}
