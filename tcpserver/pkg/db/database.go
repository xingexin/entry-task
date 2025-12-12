package db

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL 驱动

	"entry-task/tcpserver/config"
	log "entry-task/tcpserver/pkg/logger"

	"go.uber.org/zap"
)

// InitDB 初始化数据库连接（使用 sqlx）
func InitDB(cfg *config.Config) (*sqlx.DB, error) {
	log.Info("开始初始化数据库连接",
		zap.String("driver", cfg.Database.Driver),
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Database),
	)

	var driverName string
	var dsn string

	// 根据驱动类型选择驱动和 DSN
	switch cfg.Database.Driver {
	case "mysql":
		driverName = "mysql"
		dsn = cfg.Database.GetDSN()
		log.Debug("使用 MySQL 驱动")

	case "postgres", "pgsql":
		driverName = "postgres"
		dsn = cfg.Database.GetDSN()
		log.Debug("使用 PostgreSQL 驱动")

	default:
		log.Error("不支持的数据库驱动", zap.String("driver", cfg.Database.Driver))
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Database.Driver)
	}

	// 打开数据库连接
	log.Debug("正在建立数据库连接...")
	db, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		log.Error("连接数据库失败",
			zap.Error(err),
			zap.String("driver", cfg.Database.Driver),
			zap.String("host", cfg.Database.Host),
		)
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	//闭包处理defer
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	// 配置连接池
	log.Debug("配置数据库连接池",
		zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
		zap.Int("conn_max_lifetime", cfg.Database.ConnMaxLifetime),
	)
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	// 测试连接
	log.Debug("测试数据库连接...")
	if err := db.Ping(); err != nil {
		log.Error("数据库连接测试失败", zap.Error(err))
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Info("数据库连接成功",
		zap.String("driver", cfg.Database.Driver),
		zap.String("database", cfg.Database.Database),
	)

	return db, nil
}
