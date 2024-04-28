package service

import (
	"encoding/json"
	"net/http"

	"metric_service/storage"

	"github.com/IBM/sarama"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

type Statistic struct {
	Login  string `json:"login"`
	TaskID string `json:"task_id"`
}

type MetricServer struct {
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
				view := storage.ViewInfo{
					UserLogin: stat.Login,
					TaskId:    stat.TaskID,
				}

				if err := server.Storage.Create(&view).Error; err != nil {
					server.Logger.Errorf("Failed to create View: %v", err)
					return err
				}
			}
			if string(message.Key) == "like" {
				like := storage.LikeInfo{
					UserLogin: stat.Login,
					TaskId:    stat.TaskID,
				}

				if err := server.Storage.Create(&like).Error; err != nil {
					server.Logger.Errorf("Failed to create View: %v", err)
					return err
				}
			}

		case err := <-partition.Errors():
			server.Logger.Printf("Ошибка во время потребления: %v\n", err)
		}
	}
}

func (server *MetricServer) Start() error {
	server.Logger.Info("starting metric service")

	server.ConfigureStorage()

	server.Logger.Info("configure db")

	server.NewConsumer()

	go server.ConsumeMessages("post_events")

	server.Logger.Info("starting listen")
	return http.ListenAndServe(server.Config.Host+server.Config.Port, server.Router)
}
