package jwt

import (
	"time"
	"github.com/golang-jwt/jwt/v5"

	"sso/internal/domain/models"
)

func NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	// Генерируем токе
	token := jwt.New(jwt.SigningMethodHS256)

	// Добавляем к токену метаданные
	claims := token.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(duration).Unix() // Когда токен протухнет
	claims["app_id"] = app.ID

	// Подписываем токен с помощью секретного ключа приложения
	tokenString, err := token.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}