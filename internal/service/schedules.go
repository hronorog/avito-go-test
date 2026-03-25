package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/hronorog/avito-go-test/internal/repo"
)

var ErrScheduleExists = errors.New("schedule already exists")

func (s *Service) CreateSchedule(
	ctx context.Context,
	roomID uuid.UUID,
	days []int,
	start, end time.Time,
) (*repo.Schedule, error) {
	if len(days) == 0 {
		return nil, ErrInvalidRequest
	}
	for _, d := range days {
		if d < 1 || d > 7 {
			return nil, ErrInvalidRequest
		}
	}
	if !start.Before(end) {
		return nil, ErrInvalidRequest
	}

	sc, err := s.repo.CreateSchedule(ctx, roomID, days, start, end)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrScheduleExists
		}
		return nil, err
	}
	return sc, nil
}

func isUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
		return true
	}
	return false
}