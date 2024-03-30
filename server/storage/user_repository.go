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

// func (r *UserRepository) UpdateUser(user *model.User, request *RequestUpdate) (*model.User, error) {
// 	var updates []string
// 	var args []interface{}

// 	if request.UserName != "" {
// 		updates = append(updates, "user_name = ?")
// 		args = append(args, request.UserName)
// 		user.UserName = request.UserName
// 	}
// 	if request.UserSurname != "" {
// 		updates = append(updates, "user_surname = ?")
// 		args = append(args, request.UserSurname)
// 		user.UserSurname = request.UserSurname
// 	}
// 	if request.UserPhoneNumber != "" {
// 		updates = append(updates, "user_phone_number = ?")
// 		args = append(args, request.UserPhoneNumber)
// 		user.UserPhoneNumber = request.UserPhoneNumber
// 	}
// 	if !request.UserBirthday.IsZero() {
// 		updates = append(updates, "user_birthday = ?")
// 		args = append(args, request.UserBirthday)
// 		user.UserBirthday = request.UserBirthday
// 	}
// 	if request.UserEmail != "" {
// 		updates = append(updates, "user_email = ?")
// 		args = append(args, request.UserEmail)
// 		user.UserEmail = request.UserEmail
// 	}

// 	if len(updates) == 0 {
// 		return nil, fmt.Errorf("no fields to update")
// 	}

// 	args = append(args, user.ID)

// 	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(updates, ", "))
// 	_, err := r.storage.DB.Exec(query, args...)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return user, nil
// }

func (r *UserRepository) UpdateUser(user *model.User, request *RequestUpdate) (*model.User, error) {
	var updates []string
	var args []interface{}
	counter := 1

	if request.UserName != "" {
		updates = append(updates, fmt.Sprintf("user_name = $%d", counter))
		args = append(args, request.UserName)
		counter++
		user.UserName = request.UserName
	}
	if request.UserSurname != "" {
		updates = append(updates, fmt.Sprintf("user_surname = $%d", counter))
		args = append(args, request.UserSurname)
		counter++
		user.UserSurname = request.UserSurname
	}
	if request.UserPhoneNumber != "" {
		updates = append(updates, fmt.Sprintf("user_phone_number = $%d", counter))
		args = append(args, request.UserPhoneNumber)
		counter++
		user.UserPhoneNumber = request.UserPhoneNumber
	}
	if !request.UserBirthday.IsZero() {
		updates = append(updates, fmt.Sprintf("user_birthday = $%d", counter))
		args = append(args, request.UserBirthday)
		counter++
		user.UserBirthday = request.UserBirthday
	}
	if request.UserEmail != "" {
		updates = append(updates, fmt.Sprintf("user_email = $%d", counter))
		args = append(args, request.UserEmail)
		counter++
		user.UserEmail = request.UserEmail
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Добавляем id пользователя в args (на последнюю позицию)
	args = append(args, user.ID)

	// Создаём строку с номером плейсхолдера для id
	idPlaceholder := fmt.Sprintf("$%d", counter)

	// Формируем SQL-запрос с использованием номера плейсхолдера для id
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = %s", strings.Join(updates, ", "), idPlaceholder)

	// Выполняем запрос с подставленными аргументами
	_, err := r.storage.DB.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	// Возвращаем обновлённый объект пользователя
	return user, nil
}
