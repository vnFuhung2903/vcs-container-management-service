package databases

import (
	"fmt"

	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/env"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgresDb(env env.PostgresEnv) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		env.PostgresHost, env.PostgresUser, env.PostgresPassword, env.PostgresName, env.PostgresPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&entities.Container{}); err != nil {
		return nil, err
	}
	return db, nil
}
