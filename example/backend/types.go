package backend

import (
	"github.com/andrewwormald/gobridge/example/backend/second"
)

type User struct {
	ID   int64
	Name string
	Role Role
	t    second.Toy
}

type Role int

const (
	RoleUnknown Role = 0
	RoleUser    Role = 1
	RoleAdmin   Role = 2
)
