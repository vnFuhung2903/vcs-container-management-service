package env

import (
	"errors"

	"github.com/spf13/viper"
)

type AuthEnv struct {
	JWTSecret string
}

type PostgresEnv struct {
	PostgresHost     string
	PostgresUser     string
	PostgresPassword string
	PostgresName     string
	PostgresPort     string
}

type LoggerEnv struct {
	Level      string
	FilePath   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
}

type Env struct {
	AuthEnv     AuthEnv
	PostgresEnv PostgresEnv
	LoggerEnv   LoggerEnv
}

func LoadEnv() (*Env, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("POSTGRES_HOST", "localhost")
	v.SetDefault("POSTGRES_USER", "postgres")
	v.SetDefault("POSTGRES_PASSWORD", "postgres")
	v.SetDefault("POSTGRES_CONTAINER_DB", "postgres")
	v.SetDefault("POSTGRES_PORT", "5432")
	v.SetDefault("ZAP_LEVEL", "info")
	v.SetDefault("ZAP_FILEPATH", "./logs/app.log")
	v.SetDefault("ZAP_MAXSIZE", 100)
	v.SetDefault("ZAP_MAXAGE", 10)
	v.SetDefault("ZAP_MAXBACKUPS", 30)

	authEnv := AuthEnv{
		JWTSecret: v.GetString("JWT_SECRET_KEY"),
	}
	if authEnv.JWTSecret == "" {
		return nil, errors.New("auth environment variables are empty")
	}

	postgresEnv := PostgresEnv{
		PostgresHost:     v.GetString("POSTGRES_HOST"),
		PostgresUser:     v.GetString("POSTGRES_USER"),
		PostgresPassword: v.GetString("POSTGRES_PASSWORD"),
		PostgresName:     v.GetString("POSTGRES_CONTAINER_DB"),
		PostgresPort:     v.GetString("POSTGRES_PORT"),
	}
	if postgresEnv.PostgresHost == "" || postgresEnv.PostgresUser == "" || postgresEnv.PostgresPassword == "" || postgresEnv.PostgresName == "" || postgresEnv.PostgresPort == "" {
		return nil, errors.New("postgres environment variables are empty")
	}

	loggerEnv := LoggerEnv{
		Level:      v.GetString("ZAP_LEVEL"),
		FilePath:   v.GetString("ZAP_FILEPATH"),
		MaxSize:    v.GetInt("ZAP_MAXSIZE"),
		MaxAge:     v.GetInt("ZAP_MAXAGE"),
		MaxBackups: v.GetInt("ZAP_MAXBACKUPS"),
	}
	if loggerEnv.Level == "" || loggerEnv.FilePath == "" || loggerEnv.MaxSize <= 0 || loggerEnv.MaxAge <= 0 || loggerEnv.MaxBackups <= 0 {
		return nil, errors.New("logger environment variables are empty or invalid")
	}

	return &Env{
		AuthEnv:     authEnv,
		PostgresEnv: postgresEnv,
		LoggerEnv:   loggerEnv,
	}, nil
}
