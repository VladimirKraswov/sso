package auth

import (
	"context"
	"errors"
	auth "sso/internal/services"
	"sso/internal/storage"

	ssov1 "github.com/VladimirKraswov/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"  // Набор статус кодов
	"google.golang.org/grpc/status" // Статус ответа
)

const (
	emptyValue = -1
)

// Описываем интерфейс в месте его использования
type Auth interface {
	Login(ctx context.Context, email string, password string, appID int) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (userID int64, err error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type serverAPI struct {
	 // Временная махинация которая позволяет запустить приложение без реализации всех методов
	 // По сути получаем временные заглушки наших методов IsAdmin, Login, Register
	ssov1.UnimplementedAuthServer
	auth Auth
}

func Register(gRPCServer *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPCServer, &serverAPI{auth: auth})
}

// req - Данные от клиента идущие в запрос. Например email := req.GetEmail()
// ssov1.LoginResponse - Сюда записываем данные которые отдадим клиенту
func (s *serverAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {
	// Валидация
	if err := validateLogin(req); err != nil {
		return nil, err
	}

	// Отрабатывает сервисный слой (слой бизнеслогики)
	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), int(req.GetAppId()))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "Invalid argument")
		}
		return nil, status.Error(codes.Internal, "Internal error")
	}

	// Отдаем клиенту ответ
	return &ssov1.LoginResponse{Token: token}, nil
}

func (s *serverAPI) Register(ctx context.Context, req *ssov1.RegisterRequest,) (*ssov1.RegisterResponse, error) {
	// Валидация
	if err := validateRegister(req); err != nil {
		return nil, err
	}

	userID, err := s.auth.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword())

	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		return nil, status.Error(codes.Internal, "Internal error")
	}

		// Отдаем клиенту ответ
		return &ssov1.RegisterResponse{UserId: userID}, nil
}

func (s *serverAPI) IsAdmin(ctx context.Context, req *ssov1.IsAdminRequest,) (*ssov1.IsAdminResponse, error) {
	// Валидация
	if err := validateIsAdmin(req); err != nil {
		return nil, err
	}

	isAdmin, err := s.auth.IsAdmin(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "Internal error")
	}

	// Отдаем клиенту ответ
	return &ssov1.IsAdminResponse{IsAdmin: isAdmin}, nil
}

func validateLogin (req *ssov1.LoginRequest) error  {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, "email is required")
	}

	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}

	if req.GetAppId() == emptyValue {
		return status.Error(codes.InvalidArgument, "app_id is required")
	}

	return nil
}

func validateRegister (req *ssov1.RegisterRequest) error  {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, "email is required")
	}

	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}

	return nil
}

func validateIsAdmin (req *ssov1.IsAdminRequest) error  {
	if req.GetUserId() == emptyValue {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	return nil
}