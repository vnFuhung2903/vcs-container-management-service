package env

import (
	"errors"

	"github.com/spf13/viper"
)

type AuthEnv struct {
	JWTSecret string `mapstructure:"JWT_SECRET_KEY"`
}

type PostgresEnv struct {
	PostgresHost     string `mapstructure:"POSTGRES_HOST"`
	PostgresUser     string `mapstructure:"POSTGRES_USER"`
	PostgresPassword string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresName     string `mapstructure:"POSTGRES_NAME"`
	PostgresPort     string `mapstructure:"POSTGRES_PORT"`
}

type LoggerEnv struct {
	Level      string `mapstructure:"ZAP_LEVEL"`
	FilePath   string `mapstructure:"ZAP_FILEPATH"`
	MaxSize    int    `mapstructure:"ZAP_MAXSIZE"`
	MaxAge     int    `mapstructure:"ZAP_MAXAGE"`
	MaxBackups int    `mapstructure:"ZAP_MAXBACKUPS"`
}

type Env struct {
	AuthEnv     AuthEnv
	PostgresEnv PostgresEnv
	LoggerEnv   LoggerEnv
}

func LoadEnv(path string) (*Env, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("POSTGRES_HOST", "localhost")
	v.SetDefault("POSTGRES_USER", "postgres")
	v.SetDefault("POSTGRES_PASSWORD", "postgres")
	v.SetDefault("POSTGRES_NAME", "postgres")
	v.SetDefault("POSTGRES_PORT", "5432")
	v.SetDefault("ZAP_LEVEL", "info")
	v.SetDefault("ZAP_FILEPATH", "./logs/app.log")
	v.SetDefault("ZAP_MAXSIZE", 100)
	v.SetDefault("ZAP_MAXAGE", 10)
	v.SetDefault("ZAP_MAXBACKUPS", 30)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var authEnv AuthEnv
	var loggerEnv LoggerEnv
	var postgresEnv PostgresEnv

	if err := v.Unmarshal(&authEnv); err != nil || authEnv.JWTSecret == "" {
		err = errors.New("auth environment variables are empty")
		return nil, err
	}
	if err := v.Unmarshal(&loggerEnv); err != nil {
		return nil, err
	}
	if err := v.Unmarshal(&postgresEnv); err != nil || postgresEnv.PostgresUser == "" || postgresEnv.PostgresName == "" || postgresEnv.PostgresHost == "" || postgresEnv.PostgresPort == "" {
		err = errors.New("posgres environment variables are empty")
		return nil, err
	}
	return &Env{
		AuthEnv:     authEnv,
		PostgresEnv: postgresEnv,
		LoggerEnv:   loggerEnv,
	}, nil
}
