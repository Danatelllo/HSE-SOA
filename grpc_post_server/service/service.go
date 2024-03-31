package service

import (
	"context"
	"errors"
	"fmt"
	pb "grpc_post_server/proto"
	"grpc_post_server/storage"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type GrpcTaskServer struct {
	pb.UnimplementedTaskServerServer
	Config  *GrpcTaskServerConfig
	Logger  *logrus.Logger
	Storage *gorm.DB
}

func NewGrpcTaskServer(config *GrpcTaskServerConfig) *GrpcTaskServer {
	return &GrpcTaskServer{
		Config: config,
		Logger: logrus.New(),
	}
}

func (server *GrpcTaskServer) ConfigureStorage() error {
	storage, err := storage.New(server.Config.StorageConfig)
	if err != nil {
		return err
	}

	server.Storage = storage

	return nil
}

func (s *GrpcTaskServer) CreateTask(ctx context.Context, in *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	Task := storage.Task{
		Login: in.Login,
		Title: in.Title,
		Body:  in.Body,
		Date:  in.Date.AsTime(),
	}

	if err := s.Storage.Create(&Task).Error; err != nil {
		s.Logger.Errorf("Failed to create Task: %v", err)
		return nil, err
	}

	return &pb.CreateTaskResponse{TaskId: fmt.Sprintf("%d", Task.ID)}, nil
}

func (s *GrpcTaskServer) DeleteTask(ctx context.Context, in *pb.DeleteTaskRequest) (*emptypb.Empty, error) {
	if err := s.Storage.Delete(&storage.Task{}, "id = ?", in.GetTaskId()).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *GrpcTaskServer) UpdateTask(ctx context.Context, in *pb.UpdateTaskRequest) (*emptypb.Empty, error) {
	var Task storage.Task
	if err := s.Storage.First(&Task, "id = ?", in.GetTaskId()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("Failed to find Task with ID: %s", in.GetTaskId())
		}
		return nil, err
	}

	if in.GetTitle() != "" {
		Task.Title = in.GetTitle()
	}
	if in.GetBody() != "" {
		Task.Body = in.GetBody()
	}
	Task.Date = in.GetDate().AsTime()
	if err := s.Storage.Save(&Task).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (server *GrpcTaskServer) Start() error {
	server.Logger.Info("starting GrpcTaskServer")

	server.Logger.Info("configure Grpc Task DB")
	if err := server.ConfigureStorage(); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", server.Config.Port)
	if err != nil {
		server.Logger.Fatalf("failed to listen: %v", err)
		return err
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	pb.RegisterTaskServerServer(grpcServer, server)
	server.Logger.Info("Server listening at", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		server.Logger.Fatalf("failed to serve: %v", err)
		return err
	}

	return nil

}

func (s *GrpcTaskServer) GetTask(ctx context.Context, in *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	var Task storage.Task
	if err := s.Storage.First(&Task, "id = ?", in.GetTaskId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Failed to find Task with ID: %s", in.GetTaskId())
		}
		s.Logger.Errorf("Failed to found: %v", err)
		return nil, err
	}

	date := timestamppb.New(Task.Date)

	return &pb.GetTaskResponse{
		Login: Task.Login,
		Title: Task.Title,
		Body:  Task.Body,
		Date:  date,
	}, nil
}

func (s *GrpcTaskServer) GetListTasks(ctx context.Context, req *pb.GetListTasksRequest) (*pb.GetListTasksResponse, error) {
	login := req.GetLogin()

	var tasks []*storage.Task
	result := s.Storage.Where("login = ?", login).Limit(int(req.GetLimit())).Offset(int(req.GetOffset())).Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}

	var taskResponses []*pb.GetTaskResponse
	for _, task := range tasks {
		taskResponse := &pb.GetTaskResponse{
			Login: task.Login,
			Title: task.Title,
			Body:  task.Body,
			Date:  timestamppb.New(task.Date),
		}
		taskResponses = append(taskResponses, taskResponse)
	}

	return &pb.GetListTasksResponse{
		Tasks: taskResponses,
	}, nil
}
