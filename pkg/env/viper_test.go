package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ViperSuite struct {
	suite.Suite
}

func TestViperSuite(t *testing.T) {
	suite.Run(t, new(ViperSuite))
}

func (suite *ViperSuite) SetupTest() {
	envVars := []string{
		"JWT_SECRET_KEY",
		"MAIL_USERNAME",
		"MAIL_PASSWORD",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_NAME",
		"ZAP_LEVEL",
		"ZAP_FILEPATH",
		"ZAP_MAXSIZE",
		"ZAP_MAXAGE",
		"ZAP_MAXBACKUPS",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func (suite *ViperSuite) createEnvVars(vars map[string]string) {
	for k, v := range vars {
		err := os.Setenv(k, v)
		suite.Require().NoError(err)
	}
}

func (suite *ViperSuite) TestLoadEnv() {
	envContent := map[string]string{
		"JWT_SECRET_KEY":    "test_jwt_secret",
		"POSTGRES_HOST":     "postgres_host",
		"POSTGRES_USER":     "test_user",
		"POSTGRES_PASSWORD": "test_db_password",
		"POSTGRES_NAME":     "test_db",
		"POSTGRES_PORT":     "5432",
		"ZAP_LEVEL":         "info",
		"ZAP_FILEPATH":      "/tmp/app.log",
		"ZAP_MAXSIZE":       "100",
		"ZAP_MAXAGE":        "30",
		"ZAP_MAXBACKUPS":    "5",
	}

	suite.createEnvVars(envContent)
	env, err := LoadEnv()
	suite.NoError(err)
	suite.NotNil(env)

	suite.Equal("test_jwt_secret", env.AuthEnv.JWTSecret)

	suite.Equal("postgres_host", env.PostgresEnv.PostgresHost)
	suite.Equal("test_user", env.PostgresEnv.PostgresUser)
	suite.Equal("test_db_password", env.PostgresEnv.PostgresPassword)
	suite.Equal("test_db", env.PostgresEnv.PostgresName)
	suite.Equal("5432", env.PostgresEnv.PostgresPort)

	suite.Equal("info", env.LoggerEnv.Level)
	suite.Equal("/tmp/app.log", env.LoggerEnv.FilePath)
	suite.Equal(100, env.LoggerEnv.MaxSize)
	suite.Equal(30, env.LoggerEnv.MaxAge)
	suite.Equal(5, env.LoggerEnv.MaxBackups)
}

func (suite *ViperSuite) TestLoadEnvPartialConfig() {
	envContent := map[string]string{
		"JWT_SECRET_KEY": "partial_secret",
		"POSTGRES_USER":  "partial_user",
	}
	suite.createEnvVars(envContent)

	env, err := LoadEnv()
	suite.NoError(err)
	suite.NotNil(env)

	suite.Equal("partial_secret", env.AuthEnv.JWTSecret)

	suite.Equal(100, env.LoggerEnv.MaxSize)
	suite.Equal(10, env.LoggerEnv.MaxAge)
	suite.Equal(30, env.LoggerEnv.MaxBackups)

	suite.Equal("partial_user", env.PostgresEnv.PostgresUser)
}

func (suite *ViperSuite) TestLoadEnvEmptyConfig() {
	envContent := map[string]string{}
	suite.createEnvVars(envContent)

	env, err := LoadEnv()
	suite.Error(err)
	suite.Nil(env)
}

func (suite *ViperSuite) TestLoadEnvInvalidLoggerValues() {
	envContent := map[string]string{
		"JWT_SECRET_KEY": "test_jwt_secret",
		"ZAP_MAXSIZE":    "invalid_number",
		"ZAP_MAXAGE":     "invalid_number",
		"ZAP_MAXBACKUPS": "invalid_number",
	}
	suite.createEnvVars(envContent)
	env, err := LoadEnv()

	suite.Error(err)
	suite.Nil(env)
}

func (suite *ViperSuite) TestLoadEnvEmptyPostgresValues() {
	envContent := map[string]string{
		"JWT_SECRET_KEY": "test_jwt_secret",
		"POSTGRES_HOST":  "",
		"POSTGRES_USER":  "",
		"POSTGRES_NAME":  "",
		"POSTGRES_PORT":  "",
	}

	suite.createEnvVars(envContent)
	env, err := LoadEnv()

	suite.Error(err)
	suite.Nil(env)
}
