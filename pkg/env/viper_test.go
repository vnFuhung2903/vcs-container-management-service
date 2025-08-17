package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ViperSuite struct {
	suite.Suite
	tempDir string
}

func (suite *ViperSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "viper_test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *ViperSuite) TearDownSuite() {
	os.RemoveAll(suite.tempDir)
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

func (suite *ViperSuite) createEnvFile(content string) string {
	envFile := filepath.Join(suite.tempDir, ".env")
	err := os.WriteFile(envFile, []byte(content), 0644)
	suite.Require().NoError(err)
	return envFile
}

func (suite *ViperSuite) TestLoadEnv() {
	envContent := `JWT_SECRET_KEY=test_jwt_secret
POSTGRES_HOST=postgres_host
POSTGRES_USER=test_user
POSTGRES_PASSWORD=test_db_password
POSTGRES_NAME=test_db
POSTGRES_PORT=5432
ZAP_LEVEL=info
ZAP_FILEPATH=/tmp/app.log
ZAP_MAXSIZE=100
ZAP_MAXAGE=30
ZAP_MAXBACKUPS=5`

	suite.createEnvFile(envContent)
	env, err := LoadEnv(suite.tempDir)
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
	envContent := `JWT_SECRET_KEY=partial_secret
POSTGRES_USER=partial_user`
	suite.createEnvFile(envContent)

	env, err := LoadEnv(suite.tempDir)
	suite.NoError(err)
	suite.NotNil(env)

	suite.Equal("partial_secret", env.AuthEnv.JWTSecret)

	suite.Equal(100, env.LoggerEnv.MaxSize)
	suite.Equal(10, env.LoggerEnv.MaxAge)
	suite.Equal(30, env.LoggerEnv.MaxBackups)

	suite.Equal("partial_user", env.PostgresEnv.PostgresUser)
}

func (suite *ViperSuite) TestLoadEnvConfigFileNotFound() {
	nonExistentPath := "/path/that/does/not/exist"
	env, err := LoadEnv(nonExistentPath)
	suite.Error(err)
	suite.Nil(env)
	suite.Contains(err.Error(), "Config File \".env\" Not Found")
}

func (suite *ViperSuite) TestLoadEnvInvalidConfigPath() {
	invalidPath := filepath.Join(suite.tempDir, "not_a_directory.txt")
	err := os.WriteFile(invalidPath, []byte("content"), 0644)
	suite.Require().NoError(err)

	env, err := LoadEnv(invalidPath)
	suite.Error(err)
	suite.Nil(env)
}

func (suite *ViperSuite) TestLoadEnvEmptyConfig() {
	suite.createEnvFile("")

	env, err := LoadEnv(suite.tempDir)
	suite.Error(err)
	suite.Nil(env)
}

func (suite *ViperSuite) TestLoadEnvInvalidLoggerValues() {
	envContent := `JWT_SECRET_KEY=test_jwt_secret
ZAP_MAXSIZE=invalid_number
ZAP_MAXAGE=also_invalid
ZAP_MAXBACKUPS=not_a_number
MAIL_USERNAME=test@example.com
MAIL_PASSWORD=test_password`

	suite.createEnvFile(envContent)
	env, err := LoadEnv(suite.tempDir)

	suite.Error(err)
	suite.Nil(env)
}

func (suite *ViperSuite) TestLoadEnvEmptyPostgresValues() {
	envContent := `JWT_SECRET_KEY=test_jwt_secret
POSTGRES_HOST=
POSTGRES_USER=
POSTGRES_NAME=
POSTGRES_PORT=
MAIL_USERNAME=test@example.com
MAIL_PASSWORD=test_password`

	suite.createEnvFile(envContent)
	env, err := LoadEnv(suite.tempDir)

	suite.Error(err)
	suite.Nil(env)
}
