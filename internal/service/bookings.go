package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hronorog/avito-go-test/internal/repo"
)

var (
	ErrSlotAlreadyBooked = errors.New("slot already booked")
	ErrSlotInPast        = errors.New("slot in past")
)

func (s *Service) CreateBooking(
	ctx context.Context,
	roomID, userID uuid.UUID,
	start, end time.Time,
) (*repo.Booking, error) {
	// проверяем длительность 30 минут
	if !start.Before(end) || end.Sub(start) != 30*time.Minute {
		return nil, ErrInvalidRequest
	}
	// в прошлом нельзя
	if end.Before(time.Now()) {
		return nil, ErrSlotInPast
	}

	slot, err := s.repo.FindOrCreateSlot(ctx, roomID, start, end)
	if err != nil {
		return nil, err
	}
	if slot.Status != "FREE" {
		return nil, ErrSlotAlreadyBooked
	}

	b, err := s.repo.BookSlot(ctx, slot.ID, userID)
	if err != nil {
		if errors.Is(err, repo.ErrSlotNotFree) {
			return nil, ErrSlotAlreadyBooked
		}
		return nil, err
	}
	return b, nil
}
