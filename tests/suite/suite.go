package suite

import (
	"context"
	"net"
	"sso/internal/config"
	"strconv"
	"testing"

	ssov1 "github.com/VladimirKraswov/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcHost = "localhost"
)

type Suite struct {
	*testing.T									// Потребуется для вызова метода *testing.T внутри Suite
	Cfg *config.Config					// Конфигурация приложения
	AuthClient ssov1.AuthClient // Клиент для взаимодействия с gRPC-сервером
}

func New(t *testing.T) (context.Context, *Suite) {
	t.Helper() // Говорим что текущая функция является хелпером. Это нужно чтобы при фейли тестов когда правильно выводился стек вызовов и эта функция не была финальной
	t.Parallel() // Говорим что мы можем выполнять тесты паралельно, это ускоряет их работу но могут возникнуть нюансы

	cfg := config.MustLoadByPath("../config/local.yaml")

	// Создаем контекст с ограниченным временем жизни
	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.GRPC.Timeout)

	// По завершению тестов нужно отменить контекст
	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	// Создаем gRPC-клиент для нашего сервиса
	cc, err := grpc.DialContext(
		context.Background(),
		grpcAddress(cfg),
		grpc.WithTransportCredentials(insecure.NewCredentials()), // Используем insecure.NewCredentials (Не безопасные соединения для тестов)
	)
	if err != nil {
		t.Fatalf("grpc server connection failed: %v", err)
	}

	return ctx, &Suite{
		T: t,
		Cfg: cfg,
		AuthClient: ssov1.NewAuthClient(cc),
	}
}

func grpcAddress(cfg *config.Config) string {
	// JoinHostPort объединяет хост и порт в общий адрес
	return net.JoinHostPort(grpcHost, strconv.Itoa(cfg.GRPC.Port))
}

