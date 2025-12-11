package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"entry-task/tcpserver/config"
	"entry-task/tcpserver/pkg/logger"
	"go.uber.org/zap"
)

// InitDB 初始化数据库连接
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	logger.Info("开始初始化数据库连接",
		zap.String("driver", cfg.Database.Driver),
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Database),
	)

	var dialector gorm.Dialector

	// 根据驱动类型选择方言
	switch cfg.Database.Driver {
	case "mysql":
		dsn := cfg.Database.GetDSN()
		dialector = mysql.Open(dsn)
		logger.Debug("使用 MySQL 驱动", zap.String("dsn", maskPassword(dsn)))

	case "postgres", "pgsql":
		dsn := cfg.Database.GetDSN()
		dialector = postgres.Open(dsn)
		logger.Debug("使用 PostgreSQL 驱动", zap.String("dsn", maskPassword(dsn)))

	default:
		logger.Error("不支持的数据库驱动", zap.String("driver", cfg.Database.Driver))
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Database.Driver)
	}

	// 配置 GORM 日志（集成 zap）
	gormConfig := &gorm.Config{
		Logger: NewGormLogger(), // 使用自定义的 zap 日志适配器
	}

	// 打开数据库连接
	logger.Debug("正在建立数据库连接...")
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		logger.Error("连接数据库失败",
			zap.Error(err),
			zap.String("driver", cfg.Database.Driver),
			zap.String("host", cfg.Database.Host),
		)
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层的 sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("获取数据库实例失败", zap.Error(err))
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 配置连接池
	logger.Debug("配置数据库连接池",
		zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
		zap.Int("conn_max_lifetime", cfg.Database.ConnMaxLifetime),
	)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	// 测试连接
	logger.Debug("测试数据库连接...")
	if err := sqlDB.Ping(); err != nil {
		logger.Error("数据库连接测试失败", zap.Error(err))
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	logger.Info("数据库连接成功",
		zap.String("driver", cfg.Database.Driver),
		zap.String("database", cfg.Database.Database),
	)

	return db, nil
}

// maskPassword 隐藏 DSN 中的密码（用于日志安全）
func maskPassword(dsn string) string {
	// 简单实现：隐藏密码
	return "***masked***"
}

// GormLogger GORM 日志适配器（将 GORM 日志输出到 zap）
type GormLogger struct {
	LogLevel gormLogger.LogLevel
}

// NewGormLogger 创建 GORM 日志适配器
func NewGormLogger() *GormLogger {
	return &GormLogger{
		LogLevel: gormLogger.Info, // 可以从配置读取
	}
}

// LogMode 设置日志级别
func (l *GormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	l.LogLevel = level
	return l
}

// Info 记录 Info 级别日志
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLogger.Info {
		logger.Sugar.Infof(msg, data...)
	}
}

// Warn 记录 Warn 级别日志
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLogger.Warn {
		logger.Sugar.Warnf(msg, data...)
	}
}

// Error 记录 Error 级别日志
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLogger.Error {
		logger.Sugar.Errorf(msg, data...)
	}
}

// Trace 记录 SQL 执行日志
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil {
		// SQL 执行出错
		logger.Error("SQL执行失败",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.String("sql", sql),
		)
	} else if elapsed > 200*time.Millisecond {
		// 慢查询
		logger.Warn("慢查询",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	} else if l.LogLevel >= gormLogger.Info {
		// 正常查询
		logger.Debug("SQL执行",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	}
}
