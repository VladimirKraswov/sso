package tests

import (
	"sso/tests/suite"
	"testing"
	"time"

	ssov1 "github.com/VladimirKraswov/protos/gen/go/sso"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	emptyAppID = 0
	appID = 1
	appSecret = "test-secret"
	passDefaultLen = 10
)

// HappyPath(удачно) Тестирование регистрации и логина пользователя
func TestRegisterLogin_Login_HappyPath(t *testing.T)  {
	// создаем наш набор
	ctx, st := suite.New(t)

	// Создадим случайный пароль и случайную секретную фразу
	email := gofakeit.Email()
	pass := randomFakePassword()

	// Тестирование регистрации
	respReg, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{Email: email, Password: pass})
	require.NoError(t, err) // Тест фейлится и дальше не идет
	// assert.Equal(t, ...) // Тест не фейлится а идет дальше, сравнивает параметры
	assert.NotEmpty(t, respReg.GetUserId()) // Тест не фейлится а идет дальше, смотрит что не пусто

	// Тестирование логина
	respLogin, err := st.AuthClient.Login(ctx, &ssov1.LoginRequest{Email: email, Password: pass, AppId: appID})
	require.NoError(t, err) // Тест фейлится и дальше не идет
	// assert.Equal(t, ...) // Тест не фейлится а идет дальше, сравнивает параметры
	assert.NotEmpty(t, respReg.GetUserId()) // Тест не фейлится а идет дальше, смотрит что не пусто

	// Засекаем когда время когда мы получили токен
	loginTime := time.Now()

	token := respLogin.GetToken() // Извлекаем токен
	require.NotEmpty(t, token) // Проверяем что токен не пустой

	// Парсим токен
	tokenParsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(appSecret), nil
	})
	require.NoError(t, err)
	// Проверяем что токен проходит валидацию
	claims, ok := tokenParsed.Claims.(jwt.MapClaims)
	require.True(t, ok) // Проверяем что токен успешно распарсился
	// Проверяем что токен содержит корректную информацию
	assert.Equal(t, respReg.GetUserId(), int64(claims["uid"].(float64))) // Берем respReg.GetUserId() и сравниваем с claims["uid"] которое приводим к float64 и потом конвертируем в int64
	assert.Equal(t, email, claims["email"].(string))
	assert.Equal(t, appID, int(claims["app_id"].(float64)))
	// проверяем время жизни токена
	const deltaSeconds = 1 // Точность в секундах
	// Проверяем что время жизни токена лежит в определенном диапазоне (дельте)
	createTime := loginTime.Add(st.Cfg.TokenTTL).Unix()
	expTime := claims["exp"].(float64)
	assert.InDelta(t, createTime, expTime, deltaSeconds)
}

func TestRegisterLogin_DuplicatedRegistration(t *testing.T) {
	ctx, st := suite.New(t)

	email := gofakeit.Email()
	pass := randomFakePassword()

	respReg, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: pass,
	})
	require.NoError(t, err)
	require.NotEmpty(t, respReg.GetUserId())

	respReg, err = st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: pass,
	})
	require.Error(t, err)
	assert.Empty(t, respReg.GetUserId())
	assert.ErrorContains(t, err, "user already exists")
}

func TestRegister_FailCases(t *testing.T) {
	ctx, st := suite.New(t)

	tests := []struct {
		name        string
		email       string
		password    string
		expectedErr string
	}{
		{
			name:        "Register with Empty Password",
			email:       gofakeit.Email(),
			password:    "",
			expectedErr: "password is required",
		},
		{
			name:        "Register with Empty Email",
			email:       "",
			password:    randomFakePassword(),
			expectedErr: "email is required",
		},
		{
			name:        "Register with Both Empty",
			email:       "",
			password:    "",
			expectedErr: "email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
				Email:    tt.email,
				Password: tt.password,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedErr)

		})
	}
}

func TestLogin_FailCases(t *testing.T) {
	ctx, st := suite.New(t)

	tests := []struct {
		name        string
		email       string
		password    string
		appID       int32
		expectedErr string
	}{
		{
			name:        "Login with Empty Password",
			email:       gofakeit.Email(),
			password:    "",
			appID:       appID,
			expectedErr: "password is required",
		},
		{
			name:        "Login with Empty Email",
			email:       "",
			password:    randomFakePassword(),
			appID:       appID,
			expectedErr: "email is required",
		},
		{
			name:        "Login with Both Empty Email and Password",
			email:       "",
			password:    "",
			appID:       appID,
			expectedErr: "email is required",
		},
		{
			name:        "Login with Non-Matching Password",
			email:       gofakeit.Email(),
			password:    randomFakePassword(),
			appID:       appID,
			expectedErr: "invalid email or password",
		},
		{
			name:        "Login without AppID",
			email:       gofakeit.Email(),
			password:    randomFakePassword(),
			appID:       emptyAppID,
			expectedErr: "app_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
				Email:    gofakeit.Email(),
				Password: randomFakePassword(),
			})
			require.NoError(t, err)

			_, err = st.AuthClient.Login(ctx, &ssov1.LoginRequest{
				Email:    tt.email,
				Password: tt.password,
				AppId:    tt.appID,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func randomFakePassword() string {
	return gofakeit.Password(true, true, true, true, false, passDefaultLen)
}