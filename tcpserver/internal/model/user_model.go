package model

import "time"

type User struct {
	ID             uint64    `db:"id"`
	Username       string    `db:"username"`
	PasswordHash   string    `db:"password_hash"`
	Nickname       string    `db:"nickname"`
	ProfilePicture string    `db:"profile_picture"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
