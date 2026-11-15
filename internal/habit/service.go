package habit

import "context"

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service {
	return &Service{repo: r}
}

func (s *Service) List(ctx context.Context, userID string) ([]Habit, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) Create(ctx context.Context, h *Habit) error {
	return s.repo.Create(ctx, h)
}
