package container

import (
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/dig"
	"gorm.io/gorm"

	"entry-task/tcpserver/config"
	"entry-task/tcpserver/pkg/db"
)

// Container 全局依赖注入容器
var Container *dig.Container

// Init 初始化依赖注入容器
func Init() error {
	Container = dig.New()

	// 注册所有依赖
	if err := registerProviders(); err != nil {
		return err
	}

	return nil
}

// registerProviders 注册所有提供者
func registerProviders() error {
	if err := Container.Provide(func(cfg *config.Config) (*gorm.DB, error) {
		return db.InitDB(cfg)
	}); err != nil {
		return err
	}
	return nil
}

// Invoke 调用函数，自动注入依赖
func Invoke(function interface{}) error {
	return Container.Invoke(function)
}
