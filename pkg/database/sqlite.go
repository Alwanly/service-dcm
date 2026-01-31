package database

import (
	"fmt"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewSQLiteDB(path string) (*gorm.DB, error) {
	if path == "" {
		path = ":memory:"
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}

func RunMigrations(db *gorm.DB) error {

	models := []interface{}{
		&models.Agent{},
		&models.Configuration{},
		&models.AgentConfig{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func SeedInitialData(db *gorm.DB) error {
	// Check if initial configuration exists
	var count int64
	if err := db.Model(&models.Configuration{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check existing configurations: %w", err)
	}

	if count == 0 {
		initialConfig := models.Configuration{
			ETag:       fmt.Sprintf("%x-%d", 1, time.Now().UnixNano()),
			ConfigData: "{}",
		}
		if err := db.Create(&initialConfig).Error; err != nil {
			return fmt.Errorf("failed to seed initial configuration: %w", err)
		}
	}

	return nil
}
