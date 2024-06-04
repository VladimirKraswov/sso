package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env 				string 					`yaml:"env" end-default:"local"`
	StoragePath string 					`yaml:"storage_path" env-required:"true"`
	TokenTTL 		time.Duration		`yaml:"token_ttl" env-required:"true"`
	GRPC 				GRPCConfig 			`yaml:"grpc"`
}

type GRPCConfig struct {
	Port 		int 					`yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

// По негласной договоренности функции которые не возвращают ошибок называются с прификсом Must
// Тогда функция будет просто паниковать, нам незачем пытаться обработать ошибку загрузки конфига, пусть программа падает
func MustLoad() *Config {
	path := fetchConfigPath()

	// Указан ли путь
	if path == "" {
		panic("Config path is empty")
	}

	return MustLoadByPath(path)
}

// Вынесем отдельно что бы можно было запускать из тестов с указанием пути
func MustLoadByPath(configPath string) *Config {
	// Есть ли файл по указанному пути
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("Config file not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("File to read config: " + err.Error())
	}

	return &cfg
}

// Функция получает путь до файла конфига из двух возможных источников
// Либо из переменной окружения, либо из флага, стоит учитывать что если указать и то и то, то приоритет у флага
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}