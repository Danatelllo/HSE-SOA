package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	// "net/http"

	// "os"
	pb "metric_service/proto"
	"metric_service/storage"

	"github.com/IBM/sarama"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"gorm.io/gorm"
)

type Statistic struct {
	Login  string `json:"login"`
	TaskID string `json:"task_id"`
	Owner  string `json:"owner"`
}

type MetricServer struct {
	pb.MetricsServiceServer
	Config        *MetricServerConfig
	Logger        *logrus.Logger
	Router        *mux.Router
	Storage       *gorm.DB
	PrivateKey    []byte
	KafkaConsumer sarama.Consumer
}

func NewMetricServer(config *MetricServerConfig) *MetricServer {
	return &MetricServer{
		Config: config,
		Logger: logrus.New(),
		Router: mux.NewRouter(),
	}
}

func (server *MetricServer) ConfigureStorage() error {
	storage, err := storage.New(server.Config.StorageConfig)
	if err != nil {
		return err
	}

	server.Storage = storage

	return nil
}

func (server *MetricServer) NewConsumer() error {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Producer.Retry.Max = 10
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumer, err := sarama.NewConsumer([]string{server.Config.KafkaConsumer}, config)

	if err != nil {
		server.Logger.Info("Failed to start consumer")
		return err
	}

	server.KafkaConsumer = consumer
	return nil
}

func (server *MetricServer) ConsumeMessages(topic string) error {
	partition, err := server.KafkaConsumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	for {
		select {
		case message := <-partition.Messages():
			var stat Statistic
			json.Unmarshal(message.Value, &stat)
			if string(message.Key) == "view" {
				var existingLikeCount int64
				server.Storage.Model(&storage.LikeInfo{}).Where("user_login = ? AND task_id = ?", stat.Login, stat.TaskID).Count(&existingLikeCount)
				server.Logger.Info(stat.Login, stat.TaskID)
				if existingLikeCount > 0 {
					server.Logger.Info("View already exists from this user for this task")
				} else {
					view := storage.ViewInfo{
						UserLogin: stat.Login,
						TaskId:    stat.TaskID,
					}

					if err := server.Storage.Create(&view).Error; err != nil {
						server.Logger.Errorf("Failed to create View: %v", err)
						return err
					}
					server.Logger.Info("view", view)
				}
			}

			if string(message.Key) == "like" {
				var existingLikeCount int64
				server.Storage.Model(&storage.LikeInfo{}).Where("user_login = ? AND task_id = ?", stat.Login, stat.TaskID).Count(&existingLikeCount)
				server.Logger.Info(stat.Login, stat.TaskID)
				if existingLikeCount > 0 {
					server.Logger.Info("Like already exists from this user for this task")
				} else {
					like := storage.LikeInfo{
						UserLogin: stat.Login,
						TaskId:    stat.TaskID,
					}

					if err := server.Storage.Create(&like).Error; err != nil {
						server.Logger.Errorf("Failed to create Like: %v", err)
						return err
					}
					server.Logger.Info("like", like)
				}
			}

		case err := <-partition.Errors():
			server.Logger.Printf("Ошибка во время потребления: %v\n", err)
		}
	}
}

func (s *MetricServer) GetTaskMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	var viewsCount, likesCount int64
	taskId := req.TaskId

	if err := s.Storage.Model(&storage.ViewInfo{}).Where("task_id = ?", taskId).Count(&viewsCount).Error; err != nil {
		s.Logger.Errorf("Error while fetching views count: %v", err)
		return nil, err
	}

	if err := s.Storage.Model(&storage.LikeInfo{}).Where("task_id = ?", taskId).Count(&likesCount).Error; err != nil {
		s.Logger.Errorf("Error while fetching likes count: %v", err)
		return nil, err
	}

	return &pb.MetricsResponse{
		Views: int32(viewsCount),
		Likes: int32(likesCount),
	}, nil
}

func (s *MetricServer) GetTopTasks(ctx context.Context, req *pb.TopRequest) (*pb.TopResponse, error) {
	var result []*pb.Task // Использование среза указателей
	metricType := req.MetricType
	tableName := "view_infos" // по умолчанию считаем просмотры
	if metricType == "likes" {
		tableName = "like_infos"
	}

	query := fmt.Sprintf("SELECT task_id, COUNT(*) as count FROM %s GROUP BY task_id ORDER BY COUNT(*) DESC LIMIT 5", tableName)
	rows, err := s.Storage.Raw(query).Rows()
	if err != nil {
		s.Logger.Errorf("Error while fetching top posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t pb.Task
		if err := rows.Scan(&t.Id, &t.Count); err != nil {
			s.Logger.Errorf("Error while scanning top posts: %v", err)
			return nil, err
		}
		result = append(result, &t) // Добавляем адрес t в срез
	}

	return &pb.TopResponse{TopItems: result}, nil
}

func (s *MetricServer) GetUserRanking(ctx context.Context, req *pb.UserRankingRequest) (*pb.UserRankingResponse, error) {
	var users []*pb.UserRanking // Изменение на срез указателей
	query := fmt.Sprintf("SELECT user_login, COUNT(*) as likes_count FROM like_infos GROUP BY user_login ORDER BY COUNT(*) DESC LIMIT 3")
	rows, err := s.Storage.Raw(query).Rows()
	if err != nil {
		s.Logger.Errorf("Error while fetching user ranking: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u pb.UserRanking
		if err := rows.Scan(&u.UserLogin, &u.LikesCount); err != nil {
			s.Logger.Errorf("Error while scanning user ranking: %v", err)
			return nil, err
		}
		users = append(users, &u) // Добавляем адрес u в срез
	}

	return &pb.UserRankingResponse{Users: users}, nil
}

func (server *MetricServer) Start() error {
	server.Logger.Info("starting metric service")

	server.ConfigureStorage()

	server.Logger.Info("configure db")

	if err := server.NewConsumer(); err != nil {
		server.Logger.Info(err)
	}

	go server.ConsumeMessages("post_events")

	lis, _ := net.Listen("tcp", server.Config.Port)

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	pb.RegisterMetricsServiceServer(grpcServer, server)
	server.Logger.Info("Server listening at", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		server.Logger.Fatalf("failed to serve: %v", err)
		return err
	}
	return nil
	// server.Logger.Info("starting listen")
	// return http.ListenAndServe(server.Config.Host+server.Config.Port, server.Router)
}
