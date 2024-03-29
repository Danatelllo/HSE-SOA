package service

import (
	"context"
	"errors"
	"fmt"
	pb "grpc_post_server/proto"
	"grpc_post_server/storage"
	"net"

	"github.com/golang/protobuf/ptypes"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type GrpcPostServer struct {
	Config  *GrpcPostServerConfig
	Logger  *logrus.Logger
	Router  *mux.Router
	Storage *gorm.DB
	pb.UnimplementedPostServerServer
}

func NewRestApiServer(config *GrpcPostServerConfig) *GrpcPostServer {
	return &GrpcPostServer{
		Config: config,
		Logger: logrus.New(),
		Router: mux.NewRouter(),
	}
}

func (server *GrpcPostServer) ConfigureStorage() error {
	storage, err := storage.New(server.Config.StorageConfig)
	if err != nil {
		return err
	}

	server.Storage = storage

	return nil
}

func (s *GrpcPostServer) CreatePost(ctx context.Context, in *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	post := storage.Post{
		Login: in.Login,
		Title: in.Title,
		Body:  in.Body,
		Date:  in.Date.AsTime(),
	}

	if err := s.Storage.Create(&post).Error; err != nil {
		s.Logger.Errorf("Failed to create post: %v", err)
		return nil, err
	}

	return &pb.CreatePostResponse{PostId: fmt.Sprintf("%d", post.ID)}, nil
}

func (s *GrpcPostServer) DeletePost(ctx context.Context, in *pb.DeletePostRequest) (*emptypb.Empty, error) {
	if err := s.Storage.Delete(&storage.Post{}, "id = ?", in.GetPostId()).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *GrpcPostServer) UpdatePost(ctx context.Context, in *pb.UpdatePostRequest) (*emptypb.Empty, error) {
	var post storage.Post
	if err := s.Storage.First(&post, "id = ?", in.GetPostId()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("Failed to find post with ID: %s", in.GetPostId())
		}
		return nil, err
	}

	if in.GetTitle() != "" {
		post.Title = in.GetTitle()
	}
	if in.GetBody() != "" {
		post.Body = in.GetBody()
	}
	post.Date = in.GetDate().AsTime()
	if err := s.Storage.Save(&post).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (server *GrpcPostServer) Start() error {
	server.Logger.Info("starting GrpcPostServer")

	server.Logger.Info("configure Grpc Post DB")
	if err := server.ConfigureStorage(); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", server.Config.Port)
	if err != nil {
		server.Logger.Fatalf("failed to listen: %v", err)
		return err
	}

	s := grpc.NewServer()
	pb.RegisterPostServerServer(s, &GrpcPostServer{})
	server.Logger.Info("Server listening at", lis.Addr())
	if err := s.Serve(lis); err != nil {
		server.Logger.Fatalf("failed to serve: %v", err)
		return err
	}
	return nil
}

func (s *GrpcPostServer) GetPost(ctx context.Context, in *pb.GetPostRequest) (*pb.GetPostResponse, error) {
	var post storage.Post
	if err := s.Storage.First(&post, "id = ?", in.GetPostId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Failed to find post with ID: %s", in.GetPostId())
		}
		s.Logger.Errorf("Failed to found: %v", err)
		return nil, err
	}

	date, err := ptypes.TimestampProto(post.Date)
	if err != nil {
		s.Logger.Errorf("Failed to convert time to proto time: %v", err)
		return nil, err
	}

	return &pb.GetPostResponse{
		Login: post.Login,
		Title: post.Title,
		Body:  post.Body,
		Date:  date,
	}, nil
}
