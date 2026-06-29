package entity

import (
	"github.com/beevik/guid"
	"time"
)

type User struct {
	ID        guid.Guid
	CreatedAt time.Time
	Name      string
}

func NewUser(name string) *User {
	return &User{
		Name: name,
	}
}
