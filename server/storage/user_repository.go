package storage

import (
	"errors"
	"fmt"
	"main_service/model"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type RequestUpdate struct {
	UserEmail       string    `json:"email"`
	UserName        string    `json:"name"`
	UserPassword    string    `json:"password"`
	UserSurname     string    `json:"surname"`
	UserLogin       string    `json:"login"`
	UserPhoneNumber string    `json:"phonenumber"`
	Token           string    `json:"token"`
	UserBirthday    time.Time `json:"birthday"`
}

type RequestRegister struct {
	UserEmail       string    `json:"email"`
	UserName        string    `json:"name"`
	UserPassword    string    `json:"password"`
	UserSurname     string    `json:"surname"`
	UserLogin       string    `json:"login"`
	UserPhoneNumber string    `json:"phonenumber"`
	UserBirthday    time.Time `json:"birthday"`
}

type LoginRequest struct {
	UserLogin    string `json:"login"`
	UserPassword string `json:"password"`
}

type UserRepository struct {
	storage *Storage
}

func EncryptPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("empty password")
	}

	bytePassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	return string(bytePassword), nil
}

func (r *UserRepository) Create(u *model.User) (*model.User, error) {
	err := u.Validate()
	if err != nil {
		return nil, err
	}
	encryptPassword, err := EncryptPassword(u.UserPassword)
	if err != nil {
		return nil, err
	}
	u.UserPassword = encryptPassword
	if err := r.storage.DB.QueryRow(
		"INSERT INTO users (user_email, user_password, user_name, user_surname, user_phone_number, user_login) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		u.UserEmail,
		u.UserPassword,
		u.UserName,
		u.UserSurname,
		u.UserPhoneNumber,
		u.UserLogin,
	).Scan(&u.ID); err != nil {
		return nil, err
	}

	return u, nil
}

func (r *UserRepository) FindUserByLogin(login string) (*model.User, error) {
	user := &model.User{}
	if err := r.storage.DB.QueryRow(
		"SELECT id, user_email, user_password, user_name, user_surname, user_phone_number, user_login FROM users WHERE user_login = $1", login,
	).Scan(&user.ID, &user.UserEmail, &user.UserPassword, &user.UserName, &user.UserSurname, &user.UserPhoneNumber, &user.UserLogin); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) UpdateUser(user *model.User, request *RequestUpdate) (*model.User, error) {
	var updates []string
	var args []interface{}

	if user.UserName != "" {
		updates = append(updates, "user_name = ?")
		args = append(args, user.UserName)
	}
	if user.UserSurname != "" {
		updates = append(updates, "user_surname = ?")
		args = append(args, user.UserSurname)
	}
	if user.UserPhoneNumber != "" {
		updates = append(updates, "user_phone_number = ?")
		args = append(args, user.UserPhoneNumber)
	}
	if !user.UserBirthday.IsZero() {
		updates = append(updates, "user_birthday = ?")
		args = append(args, user.UserBirthday)
	}
	if user.UserEmail != "" {
		updates = append(updates, "user_email = ?")
		args = append(args, user.UserEmail)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	args = append(args, user.ID)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(updates, ", "))
	_, err := r.storage.DB.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return user, nil
}
