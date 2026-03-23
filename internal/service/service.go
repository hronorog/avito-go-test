package service

import (
	"context"

	"github.com/hronorog/avito-go-test/internal/repo"
)

type Service struct {
	repo *repo.Repo
}

func New(r *repo.Repo) *Service {
	return &Service{repo: r}
}

func (s *Service) Health(ctx context.Context) error {
	return s.repo.Ping(ctx)
}