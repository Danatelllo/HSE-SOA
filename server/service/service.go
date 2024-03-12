package service

import (
	"encoding/json"
	"errors"
	"io"
	"main_service/model"
	"main_service/storage"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/swaggo/http-swagger/v2"
)

type RestApiServer struct {
	Config     *RestApiConfig
	Logger     *logrus.Logger
	Router     *mux.Router
	Storage    *storage.Storage
	PrivateKey []byte
}

type Claims struct {
	Userlogin string `json:"username"`
	jwt.RegisteredClaims
}

func NewRestApiServer(config *RestApiConfig) *RestApiServer {
	return &RestApiServer{
		Config: config,
		Logger: logrus.New(),
		Router: mux.NewRouter(),
	}
}

func (server *RestApiServer) Start() error {
	server.Logger.Info("starting RestApiServer")

	server.Logger.Info("configure db")
	if err := server.configureStorage(); err != nil {
		return err
	}

	server.Logger.Info("Read key")
	file, err := os.Open("private.pem")
	if err != nil {
		return err
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	file.Close()

	server.PrivateKey = bytes

	server.Register()

	server.Logger.Info("starting listen")
	return http.ListenAndServe(server.Config.Host+server.Config.Port, server.Router)
}

func (server *RestApiServer) configureStorage() error {
	storage := storage.New(server.Config.StorageConfig)
	if err := storage.ConnnectToStorage(); err != nil {
		return err
	}

	server.Storage = storage

	return nil
}

func (server *RestApiServer) Register() {
	server.Router.PathPrefix("/docs/").Handler(http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs"))))

	// Настроим Swagger UI соответствующим образом
	server.Router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8081/docs/swagger.json"), // URL указывает на расположение файла doc.json
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	server.Router.HandleFunc("/", server.HandleHello()).Methods("GET")
	server.Router.HandleFunc("/user_login", server.HandleUserLogin()).Methods("POST")
	server.Router.HandleFunc("/register_user", server.HandleRegisterUser()).Methods("POST")
	server.Router.HandleFunc("/update_user", server.HandleUpdateUser()).Methods("PUT")
}

func (server *RestApiServer) HandleHello() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello")
	}
}

func FillUserFromRequest(req *storage.RequestRegister) *model.User {
	return &model.User{
		UserEmail:       req.UserEmail,
		UserName:        req.UserName,
		UserPassword:    req.UserPassword,
		UserSurname:     req.UserSurname,
		UserPhoneNumber: req.UserPhoneNumber,
		UserBirthday:    req.UserBirthday,
		UserLogin:       req.UserLogin,
	}
}

// @Summary Вход для пользователя
// @Description Вход для пользователя с login и password.
// @ID user-login
// @Accept  json
// @Produce  json
// @Param   user  body storage.LoginRequest  true  "Информация для входа"
// @Success 201  token   "Пользователь авторизован"
// @Failure 400  "Паспорт не совпадает"
// @Failure 400  "Логин не найден"
// @Failure 500  "Внутренняя ошибка сервера"
// @Router /user_login [post]
// @Example Request { "json" : {"login": "userlogin", "password": "password123"} }
func (server *RestApiServer) HandleUserLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.LoginRequest{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}
		user, err := server.Storage.User().FindUserByLogin(req.UserLogin)
		if err != nil {
			server.HandleError(w, r, http.StatusBadRequest, errors.New("login not found"))
			return
		}

		if !user.IsPasswordEqual(req.UserPassword) {
			server.HandleError(w, r, http.StatusBadRequest, errors.New("bad password"))
			return
		}

		expirationTime := time.Now().Add(30 * time.Minute)
		claims := &Claims{
			Userlogin: user.UserLogin,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		print(token, server.PrivateKey)
		tokenString, err := token.SignedString(server.PrivateKey)
		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		server.HandleRespond(w, r, http.StatusOK, map[string]string{"token": tokenString})

	}
}

// @Summary Регистрация нового пользователя
// @Description Регистрация нового пользователя с вводом обязательных и необязательных данных.
// @ID register-user
// @Accept  json
// @Produce  json
// @Param   user  body storage.RequestRegister  true  "Информация о пользователе"
// @Success 201  {object}   model.User  "Пользователь успешно создан"
// @Failure 400  "Пользователь с таким логином или email уже существует"
// @Failure 400  "Не проходит валидация пользователя"
// @Router /register_user [post]
// @Example Request { "json" : { "email": "user@example.com", "login": "userlogin", "password": "password123", "name": "John", "surname": "Doe", "phonenumber": "1234567890", "token": "youraccesstoken", "birthday": "1990-01-01T00:00:00Z" } }
func (server *RestApiServer) HandleRegisterUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.RequestRegister{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		user, _ := server.Storage.User().FindUserByLogin(req.UserEmail)

		if user != nil {
			server.HandleError(w, r, http.StatusBadRequest, errors.New("user already exists"))
			return
		}

		user = FillUserFromRequest(req)
		if _, err := server.Storage.User().Create(user); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		server.HandleRespond(w, r, http.StatusCreated, user)
	}
}

func (server *RestApiServer) HandleUpdateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.RequestUpdate{}
		print("puk")
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, errors.New("Can not parse request"))
			return
		}
		if req.UserLogin != "" || req.UserPassword != "" {
			server.HandleError(w, r, http.StatusForbidden, errors.New("Change password or login deprecated"))
			return
		}
		print("puk")
		claims := &Claims{}

		tkn, err := jwt.ParseWithClaims(req.Token, claims, func(token *jwt.Token) (any, error) {
			return server.PrivateKey, nil
		})

		if err != nil {
			server.HandleError(w, r, http.StatusBadRequest, errors.New("bad token"))
			return
		}

		if !tkn.Valid {
			server.HandleError(w, r, http.StatusBadRequest, errors.New("bad token"))
			return
		}
		print("puk")

		user, err := server.Storage.User().FindUserByLogin(req.UserLogin)
		if err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}
		server.Storage.User().UpdateUser(user, req)
		server.HandleRespond(w, r, http.StatusOK, user)
	}
}

func (server *RestApiServer) HandleError(w http.ResponseWriter, r *http.Request, code int, err error) {
	server.HandleRespond(w, r, code, map[string]string{"error": err.Error()})
}

func (server *RestApiServer) HandleRespond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
