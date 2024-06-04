package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"sso/internal/domain/models"
	"sso/internal/lib/jwt"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage"
)

type Auth struct {
	log *slog.Logger
	userSaver 		UserSaver
	userProvider UserProvider
	appProvider 	AppProvider
	tokenTTL 		time.Duration

}

type UserSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte) (uid int64, err error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int) (models.App, error)
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists = errors.New("user already exists")
	ErrInvalidAppID = errors.New("invalid app id")
	ErrUserNotFound = errors.New("user not found")
)

// New возвращает новый экземпляр службы аутентификации.
func New(log *slog.Logger, userSaver UserSaver, userProvider UserProvider, appProvider AppProvider, tokenTTL time.Duration) *Auth {
	return &Auth{
		log: 					log,
		userSaver: 		userSaver,
		userProvider: userProvider,
		appProvider: 	appProvider,
		tokenTTL: 		tokenTTL,
	}
}

// Login проверяет, существует ли в системе пользователь с указанными учетными данными, и возвращает токен доступа.
//
// Если пользователь существует, но пароль неверен, возвращает ошибку.
// Если пользователь не существует, возвращает ошибку.
func (a *Auth) Login(ctx context.Context, email string, password string, appID int) (string, error) {
	const op = "auth.Login"

	log := a.log.With(slog.String("op", op), slog.String("email", email)) // Внимание email это GDPR данные, лучше их не логировать
	log.Info("Login user")

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", sl.Err(err))

			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		a.log.Error("failed to get user", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	// Теперь с помощью bcrypt проверим правильный ли пароль ввел юзер
	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	// Теперь смотрим в какое приложение пользователь хочет залогинится
	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op)
	}

	// Каждый токен подписывается ключем, но ключей может быть много, у каждого приложения свой
	// Пока что мы будем хранить секретный ключ приложения в БД, что не очень хорошо
	// Для получения токена мы используем ключ приложения в которое хочет залогинится пользователь
	log.Info("User logged in successfully")

	token, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate token", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op)
	}


	return token, nil
}

// RegisterNewUser регистрирует нового пользователя в системе и возвращает идентификатор пользователя.
// Если пользователь с указанным именем пользователя уже существует, возвращает ошибку.
func (a *Auth) RegisterNewUser(ctx context.Context, email string, pass string) (int64, error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(slog.String("op", op), slog.String("email", email)) // Внимание email это GDPR данные, лучше их не логировать
	log.Info("Registering user")
	// Пароль в открытом виде хранить нельзя, перед сохранением пароля в БД нам нужно его захэшировать
	// Потом при логине мы будем сравнивать один хэш с другим
	// Если же наши хэши утекут из БД то злоумышленники простые пароли все же могут из хэша восстановить, поэтому подсолим пароль
	// Подсолить значит добавить к паролю рандомную фразу. Бывают более продвинутые технологии, например динамическая соль
	// Все это мы реализуем с помощью bcrypt из golang.org/x/crypto
	// bcrypt.GenerateFromPassword - создает соль для пароля и хэширует его, делает она это bcrypt.DefaultCost количество раз
	passHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Вызываем метол SaveUser для сохранения пользователя в БД
	id, err := a.userSaver.SaveUser(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("User already exists")
			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("User registered")
	return id, nil

}

// IsAdmin checks if user is admin.
func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "auth.IsAdmin"

	log := a.log.With(slog.String("op", op), slog.Int64("user_id", userID)) // Внимание email это GDPR данные, лучше их не логировать
	log.Info("Checking if user is admin")

	isAdmin, err := a.userProvider.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			log.Warn("App not found")
			return false, fmt.Errorf("%s: %w", op, ErrInvalidAppID)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Checked is user is admin", slog.Bool("is_admin", isAdmin))

	return isAdmin, nil
}

// GDPR — новые правила обработки персональных данных в Европе для международного IT-рынка
// То-есть если проект европейский пользователь может попросить удалить все данные о нем и из логов удалить это будет не просто 
