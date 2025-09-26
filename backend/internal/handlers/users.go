package handlers

import (
	"context"
	"seer/internal/repos"

	"github.com/google/uuid"
)

type UserRepository interface {
	GetBySubProvider(ctx context.Context, sub string, provider repos.AuthProvider) (*repos.User, error)
	Insert(ctx context.Context, user *repos.User) error
	CompleteProfile(ctx context.Context, userID uuid.UUID, username string, version int64) error
	EmailTaken(ctx context.Context, email string) (bool, error)
	UsernameTaken(ctx context.Context, username string) (bool, error)
	GetByEmail(ctx context.Context, email string) (*repos.User, error)
}

type UserHandler struct {
	UserRepo UserRepository
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}
