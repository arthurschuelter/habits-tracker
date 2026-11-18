package habit_entry

import (
	"context"
)

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service {
	return &Service{repo: r}
}

func (s *Service) Create(ctx context.Context, userID string, e *HabitEntry) (HabitEntry, error) {
	entry, err := Validate(*e)
	if err != nil {
		return HabitEntry{}, err
	}

	return s.repo.Create(ctx, userID, &entry)
}
