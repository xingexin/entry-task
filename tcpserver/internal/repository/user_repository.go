package repository

import "gorm.io/gorm"

type userReader interface {
	readProfilePicture() (string, error)
	readNickname() (string, error)
}

type userWriter interface {
	uploadProfilePicture() error
	updateNickname() error
}

type UserRepo interface {
	userReader
	userWriter
}

type gormUserRepository struct {
	gormDB *gorm.DB
}

func NewGormUserRepository(gormDB *gorm.DB) UserRepo {
	return &gormUserRepository{gormDB: gormDB}
}

func (repo *gormUserRepository) readProfilePicture() (string, error) {

}

func (repo *gormUserRepository) readNickname() (string, error) {

}

func (repo *gormUserRepository) uploadProfilePicture() error {

}

func (repo *gormUserRepository) updateNickname() error {

}
