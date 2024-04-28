package storage

import (
	"errors"
	"server/model"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

type CreateTask struct {
	Title string    `json:"title"`
	Body  string    `json:"body"`
	Date  time.Time `json:"-"`
	Token string    `json:"token"`
}

type GetTaskRequest struct {
	TaskId string `json:"task_id"`
	Token  string `json:"token"`
}

type GetTaskResponse struct {
	Login string    `json:"login"`
	Title string    `json:"title"`
	Body  string    `json:"body"`
	Date  time.Time `json:"-"`
}

type DeleteTaskRequest struct {
	TaskId string `json:"task_id"`
	Token  string `json:"token"`
}

type UpdateTaskRequest struct {
	TaskId string    `json:"task_id"`
	Title  string    `json:"title"`
	Body   string    `json:"body"`
	Date   time.Time `json:"-"`
	Token  string    `json:"token"`
}

type GetListTasks struct {
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
	Token  string `json:"token"`
}

type CreateTaskResponse struct {
	TaskId string `json:"task_id"`
}

type ViewTaskRequest struct {
	TaskId string `json:"task_id"`
	Token  string `json:"token"`
}

type UserRepository struct {
	DB *gorm.DB
}

func EncryptPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("empty password")
	}

	bytePassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

	if err := r.DB.Create(u).Error; err != nil {
		return nil, err
	}

	return u, nil
}

func (r *UserRepository) FindUserByLogin(login string) (*model.User, error) {
	user := &model.User{}
	err := r.DB.Where("user_login = ?", login).First(user).Error

	return user, err
}

func (r *UserRepository) UpdateUser(user *model.User, request *RequestUpdate) (*model.User, error) {
	updateData := model.User{
		UserName:        request.UserName,
		UserSurname:     request.UserSurname,
		UserPhoneNumber: request.UserPhoneNumber,
		UserEmail:       request.UserEmail,
		UserBirthday:    request.UserBirthday,
	}

	if err := r.DB.Model(user).Updates(updateData).Error; err != nil {
		return nil, err
	}

	return user, nil
}
