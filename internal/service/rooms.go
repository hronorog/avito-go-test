package service

import (
	"context"

	"github.com/hronorog/avito-go-test/internal/repo"
)

func (s *Service) CreateRoom(ctx context.Context, name string, description *string, capacity *int) (*repo.Room, error) {
	if name == "" {
		return nil, ErrInvalidRequest
	}
	return s.repo.CreateRoom(ctx, name, description, capacity)
}

func (s *Service) ListRooms(ctx context.Context) ([]repo.Room, error) {
	return s.repo.ListRooms(ctx)
}
