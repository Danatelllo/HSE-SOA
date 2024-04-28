package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"server/model"
	"server/storage"
	"time"

	"github.com/IBM/sarama"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/swaggo/http-swagger/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	pb "server/proto"
)

type RestApiServer struct {
	Config               *RestApiConfig
	Logger               *logrus.Logger
	Router               *mux.Router
	Storage              *storage.Storage
	PrivateKey           []byte
	TaskServerGrpcClient pb.TaskServerClient
	KafkaProducer        sarama.SyncProducer
}

type KafkaMessage struct {
	Key       string
	UserLogin string
	TaskId    string
}

type Statistic struct {
	Login  string `json:"login"`
	TaskID string `json:"task_id"`
}

type Claims struct {
	UserLogin string `json:"username"`
	jwt.RegisteredClaims
}

func (server *RestApiServer) GetLoginFromToken(token string, privateKey []byte) (string, error) {
	claims := &Claims{}

	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return privateKey, nil
	})

	if err != nil {
		return "", err
	}

	if !tkn.Valid {
		return "", errors.New("bad token")
	}
	return claims.UserLogin, nil
}

func (server *RestApiServer) EnsureLogin(token string, privateKey []byte) (string, error) {
	login, err := server.GetLoginFromToken(token, privateKey)
	if err != nil {
		return "", err
	}

	_, err = server.Storage.User().FindUserByLogin(login)
	if err != nil {
		return "", err
	}
	return login, nil
}

func NewRestApiServer(config *RestApiConfig) *RestApiServer {
	return &RestApiServer{
		Config: config,
		Logger: logrus.New(),
		Router: mux.NewRouter(),
	}
}

func NewKafkaProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 10
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewRandomPartitioner

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return producer, nil
}

func (server *RestApiServer) ProduceMessage(message KafkaMessage) error {
	topic := "post_events"

	jsonData, _ := json.Marshal(Statistic{
		TaskID: message.TaskId,
		Login:  message.UserLogin,
	})

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(message.Key),
		Value: sarama.ByteEncoder(jsonData),
	}

	_, _, err := server.KafkaProducer.SendMessage(msg)
	if err != nil {
		server.Logger.Info("Produce failed")
		return err
	}

	return nil
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

	server.KafkaProducer, err = NewKafkaProducer(server.Config.KafkaConfig)
	if err != nil {
		server.Logger.Info("Failed to create producer")
		return err
	}

	server.Logger.Info("Trying to connnect task server")
	conn, err := grpc.Dial(server.Config.TaskServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return err
	}

	server.Logger.Info("Trying to get task server client")
	server.TaskServerGrpcClient = pb.NewTaskServerClient(conn)

	server.Register()

	server.Logger.Info("starting listen")
	return http.ListenAndServe(server.Config.Host+server.Config.Port, server.Router)
}

func (server *RestApiServer) configureStorage() error {
	storage := storage.New(server.Config.StorageConfig)
	if err := storage.ConnectToStorage(); err != nil {
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

	server.Router.HandleFunc("/create_task", server.HandleCreateTask()).Methods("POST")
	server.Router.HandleFunc("/get_list_tasks", server.HandleListTasks()).Methods("POST")
	server.Router.HandleFunc("/", server.HandleHello()).Methods("GET")
	server.Router.HandleFunc("/get_task", server.HandleGetTask()).Methods("POST")
	server.Router.HandleFunc("/delete_task", server.HandleDeleteTask()).Methods("PUT")
	server.Router.HandleFunc("/update_task", server.HandleUpdateTask()).Methods("PUT")
	server.Router.HandleFunc("/user_login", server.HandleUserLogin()).Methods("POST")
	server.Router.HandleFunc("/register_user", server.HandleRegisterUser()).Methods("POST")
	server.Router.HandleFunc("/update_user", server.HandleUpdateUser()).Methods("PUT")
	server.Router.HandleFunc("/view_task", server.HandleTaskView()).Methods("POST")
	// server.Router.HandleFunc("/like_task", server.HandleLikeTask()).Methods("POST")
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
// @Param   user  body storage.LoginRequest  true  "Информация для входа. Время в формате 1990-01-01T00:00:00Z"
// @Success 201  token   "Пользователь авторизован"
// @Failure 400  "Паспорт не совпадает"
// @Failure 400  "Логин не найден"
// @Failure 500  "Внутренняя ошибка сервера"
// @Router /user_login [post]
// @Example Request { "login": "userlogin", "password": "password123" }
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
			UserLogin: user.UserLogin,
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
// @Param   user  body storage.RequestRegister  true  "Информация о пользователе. Время в формате 1990-01-01T00:00:00Z"
// @Success 201  {object}   model.User  "Пользователь успешно создан"
// @Failure 400  "Пользователь с таким логином или email уже существует"
// @Failure 400  "Не проходит валидация пользователя"
// @Router /register_user [post]
func (server *RestApiServer) HandleRegisterUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.RequestRegister{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		user, err := server.Storage.User().FindUserByLogin(req.UserLogin)

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			server.HandleError(w, r, http.StatusBadRequest, errors.New("user already exists"))
		}

		user = FillUserFromRequest(req)
		if _, err := server.Storage.User().Create(user); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		server.HandleRespond(w, r, http.StatusCreated, user)
	}
}

// @Summary Обновление полей пользователя
// @Description Обновление полей пользователя с вводом обязательных и необязательных данных.
// @ID update-user
// @Accept  json
// @Produce  json
// @Param   user  body storage.RequestUpdate  true  "Токен и информация, которую пользователь хочет изменить. Время в формате 1990-01-01T00:00:00Z"
// @Success 201  {object}   model.User  "Данные пользователя обновлен"
// @Failure 403  "Нельзя сменить логин или email"
// @Failure 400  "Неверный токен"
// @Router /update_user [put]
func (server *RestApiServer) HandleUpdateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.RequestUpdate{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}
		if req.UserLogin != "" || req.UserPassword != "" {
			server.HandleError(w, r, http.StatusForbidden, errors.New("Change password or login deprecated"))
			return
		}
		login, err := server.GetLoginFromToken(req.Token, server.PrivateKey)
		if err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}
		user, err := server.Storage.User().FindUserByLogin(login)
		if err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}
		server.Storage.User().UpdateUser(user, req)
		server.HandleRespond(w, r, http.StatusOK, user)
	}
}

// @Summary Создание новой задачи
// @Description Создание задачи с обязательными и необязательными полями
// @ID create-task
// @Accept  json
// @Produce  json
// @Param   user  body storage.CreateTask  true  "Токен и информация для создания задачи"
// @Success 201  {object}   storage.CreateTaskResponse  "Пост создан"
// @Failure 400  "Неверный токен"
// @Failure 500  "Внутренняя ошибка сервера"
// @Router /create_task [post]
func (server *RestApiServer) HandleCreateTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := &storage.CreateTask{}
		if err := json.NewDecoder(r.Body).Decode(task); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		login, err := server.EnsureLogin(task.Token, server.PrivateKey)

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		req := &pb.CreateTaskRequest{
			Login: login,
			Title: task.Title,
			Body:  task.Body,
			Date:  timestamppb.New(task.Date),
		}

		resp, err := server.TaskServerGrpcClient.CreateTask(context.Background(), req)
		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		server.HandleRespond(w, r, http.StatusOK, &storage.CreateTaskResponse{
			TaskId: resp.TaskId,
		})
	}
}

// @Summary Получение задачи по id
// @Description Получение задачи
// @ID get-task
// @Accept  json
// @Produce  json
// @Param   user  body storage.GetTaskRequest  true  "Токен и id задачи"
// @Success 201  {object}   storage.GetTaskResponse  "пост найден"
// @Failure 400  "Неверный токен"
// @Failure 500  "Внутренняя ошибка сервера"
// @Router /get_task [post]
func (server *RestApiServer) HandleGetTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := &storage.GetTaskRequest{}
		if err := json.NewDecoder(r.Body).Decode(task); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		_, err := server.EnsureLogin(task.Token, server.PrivateKey)

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		req := &pb.GetTaskRequest{
			TaskId: task.TaskId,
		}

		resp, err := server.TaskServerGrpcClient.GetTask(context.Background(), req)
		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		server.HandleRespond(w, r, http.StatusOK, &storage.GetTaskResponse{
			Login: resp.Login,
			Title: resp.Title,
			Body:  resp.Body,
			Date:  resp.Date.AsTime(),
		})
	}
}

// @Summary Удаление задачи по id
// @Description Удаление задачи
// @ID delete-task
// @Accept  json
// @Produce  json
// @Param   user  body storage.DeleteTaskRequest  true  "Токен и id для удаления задачи"
// @Success 201  "задача удалена"
// @Failure 400  "Неверный токен"
// @Failure 500  "Внутренняя ошибка сервера"
// @Router /delete_task [put]
func (server *RestApiServer) HandleDeleteTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := &storage.DeleteTaskRequest{}
		if err := json.NewDecoder(r.Body).Decode(task); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		login, err := server.EnsureLogin(task.Token, server.PrivateKey)

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		req := &pb.DeleteTaskRequest{
			TaskId: task.TaskId,
			Login:  login,
		}

		_, err = server.TaskServerGrpcClient.DeleteTask(context.Background(), req)
		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}
	}
}

// @Summary Обновление задачи по id
// @Description Обновление задачи
// @ID update_task
// @Accept  json
// @Produce  json
// @Param   user  body storage.UpdateTaskRequest  true  "Токен и информация по задаче"
// @Success 201  "задача обновлена"
// @Failure 400  "Неверный токен"
// @Failure 500  "Внутренняя ошибка сервера"
// @Router /update_task [put]
func (server *RestApiServer) HandleUpdateTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := &storage.UpdateTaskRequest{}
		if err := json.NewDecoder(r.Body).Decode(task); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		login, err := server.EnsureLogin(task.Token, server.PrivateKey)

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		req := &pb.UpdateTaskRequest{
			TaskId: task.TaskId,
			Title:  task.Title,
			Body:   task.Body,
			Date:   timestamppb.New(task.Date),
			Login:  login,
		}

		_, err = server.TaskServerGrpcClient.UpdateTask(context.Background(), req)
		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}
	}
}

// @Summary Получение всех задач пользователя с пагинацией
// @Description Все задачи пользователя с пагинацией
// @ID get_list_tasks
// @Accept  json
// @Produce  json
// @Param   user  body storage.GetListTasks  true  "Токен и информация для пагинации"
// @Success 201  "Задачи получены"
// @Failure 400  "Неверный токен"
// @Failure 500  "Внутренняя ошибка сервера"
// @Router /get_list_tasks [post]
func (server *RestApiServer) HandleListTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := &storage.GetListTasks{}
		if err := json.NewDecoder(r.Body).Decode(task); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		login, err := server.EnsureLogin(task.Token, server.PrivateKey)

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		req := &pb.GetListTasksRequest{
			Limit:  task.Limit,
			Offset: task.Offset,
			Login:  login,
		}

		response, err := server.TaskServerGrpcClient.GetListTasks(context.Background(), req)
		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		server.HandleRespond(w, r, http.StatusOK, response)
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

// // @Summary Отправка просмотра задачи
// // @ID view-task
// // @Accept  json
// // @Produce  json
// // @Success 201  "Просмотр успешно добавлен"
// // @Failure 400  "Внутрення ошибка сервера"
// // @Failure 400  "Не проходит валидация пользователя"
// // @Router /view_task [post]
// func (server *RestApiServer) HandleTaskView() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		req := &storage.RequestRegister{}
// 		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
// 			server.HandleError(w, r, http.StatusBadRequest, err)
// 			return
// 		}

// 		user, err := server.Storage.User().FindUserByLogin(req.UserLogin)

// 		if !errors.Is(err, gorm.ErrRecordNotFound) {
// 			server.HandleError(w, r, http.StatusBadRequest, errors.New("user already exists"))
// 		}

// 		user = FillUserFromRequest(req)
// 		if _, err := server.Storage.User().Create(user); err != nil {
// 			server.HandleError(w, r, http.StatusBadRequest, err)
// 			return
// 		}

// 		server.HandleRespond(w, r, http.StatusCreated, user)
// 	}
// }

// @Summary Отправка просмотра задачи
// @ID view-task
// @Accept  json
// @Produce  json
// @Param   view_info  body storage.ViewTaskRequest  true  "Токен, id"
// @Success 200  "Просмотр успешно добавлен"
// @Failure 400  "Внутрення ошибка сервера"
// @Router /view_task [post]
func (server *RestApiServer) HandleTaskView() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.ViewTaskRequest{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		login, err := server.EnsureLogin(req.Token, server.PrivateKey)

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		err = server.ProduceMessage(KafkaMessage{
			Key:       "view",
			UserLogin: login,
			TaskId:    req.TaskId,
		})

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		server.HandleRespond(w, r, http.StatusCreated, nil)
	}
}

// @Summary Отправка просмотра задачи
// @ID like-task
// @Accept  json
// @Produce  json
// @Param   like_info  body storage.ViewTaskRequest  true  "Токен, id"
// @Success 200  "Like успешно добавлен"
// @Failure 400  "Внутрення ошибка сервера"
// @Router /view_task [post]
func (server *RestApiServer) HandleLikeTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.ViewTaskRequest{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			server.HandleError(w, r, http.StatusBadRequest, err)
			return
		}

		login, err := server.EnsureLogin(req.Token, server.PrivateKey)

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		err = server.ProduceMessage(KafkaMessage{
			Key:       "like",
			UserLogin: login,
			TaskId:    req.TaskId,
		})

		if err != nil {
			server.HandleError(w, r, http.StatusInternalServerError, err)
			return
		}

		server.HandleRespond(w, r, http.StatusCreated, nil)
	}
}
