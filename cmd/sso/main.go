// Этот файл не само приложение, это точка входа в приложение а приложение уже будет в пакете app
// То есть приложение app мы можем запустить не только отсюда но и из других мест, например из тестов

package main

import (
	"log/slog"
	"os"
	"os/signal"
	"sso/internal/app"
	"sso/internal/config"
	"syscall"
)

const (
	envLocal = "local"
	envDev = "dev"
	envProd = "prod"
)

func main()  {
	// 1 - Инициализируем объект конфига, парсим конфиг файл и превращаем в объект с конфигом
	cfg := config.MustLoad()

	// 2 - Инициализируем логер
	log := setupLogger(cfg.Env)

	log.Info("Start Application", 
	slog.String("env", cfg.Env),
	slog.Any("cfg", cfg),
	slog.Int("port", cfg.GRPC.Port),
)

	// 3 - Инициализация приложения (app)
	application := app.New(log, cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL)
	// 4 - Запустим наш сервер в отдельной горутине, 
	// пока мы будем в низу ждать записи в канал stop, эта рутина будет обрабатывать запросы
	go application.GRPCSrv.MustRun()

	// Слушаем сигналы ОС для реализации Graceful shutdown
	stop := make(chan os.Signal, 1) // Создаем канал в который будем писать сигналы ОС
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT) // Ждем сигналы SIGTERM, SIGINT и записываем их в канал

	sysSignal := <- stop // Зависаем на этой строчке, так как при чтении из пустого канала мы блокируемся пока в него что-то не запишется

	// ОС послала нам один из сигналов SIGTERM или SIGINT, мы записали его в канал и прочитали в sysSignal
	log.Info("Stopping application", slog.String("signal", sysSignal.String()))
	// Корректно завершаем работу сервера
	application.GRPCSrv.Stop()

	log.Info("Application stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		// Писать логи будем в os.Stdout в виде текста NewTextHandler и уровень логирования LevelDebug, что значит выводить все логи
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		// Писать логи будем в os.Stdout в виде JSON NewJSONHandler и уровень логирования LevelDebug, что значит выводить все логи
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		// Писать логи будем в os.Stdout в виде JSON NewJSONHandler и уровень логирования LevelInfo, что значит выводить начиная с info
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}