package database

import (
	"fmt"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectMySQL opens a MySQL connection with production-grade pool settings.
func ConnectMySQL() {
	cfg := config.App.MySQL

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=Asia%%2FKolkata",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Warn),
	})
	if err != nil {
		panic(fmt.Errorf("mysql connection failed: %w", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("could not get sql.DB: %w", err))
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeHours) * time.Hour)

	DB = db
	logger.Log.Info("MySQL connected successfully")
}
