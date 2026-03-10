package threats

import (
	"context"
	"strings"

	domain "admin-service/internal/domain/model"
	svcerrors "admin-service/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var allowedSeverities = map[string]struct{}{
	"critical": {},
	"high":     {},
	"medium":   {},
	"low":      {},
}

func normalizeSeverity(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "", false
	}
	if _, ok := allowedSeverities[normalized]; !ok {
		return "", false
	}
	return normalized, true
}

type Service struct {
	repo Repository
	log  *zap.Logger
}

type CreateThreatInput struct {
	Title       string
	Type        string
	Severity    string
	Indicator   string
	Description string
	CreatedBy   uuid.UUID
}

type UpdateThreatInput struct {
	Title       *string
	Type        *string
	Severity    *string
	Indicator   *string
	Description *string
}

func NewService(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo: repo,
		log:  log.Named("threats-service"),
	}
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.Threat, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	s.log.Debug("listing threats", zap.Int("limit", limit), zap.Int("offset", offset))
	threats, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		s.log.Error("repository List failed", zap.Error(err))
		return nil, err
	}
	return threats, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.Threat, error) {
	s.log.Debug("getting threat", zap.Stringer("threat_id", id))

	threat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("repository GetByID failed", zap.Stringer("threat_id", id), zap.Error(err))
		return nil, err
	}
	if threat == nil {
		s.log.Info("threat not found", zap.Stringer("threat_id", id))
		return nil, svcerrors.ErrNotFound
	}
	return threat, nil
}

func (s *Service) Create(ctx context.Context, input CreateThreatInput) (*domain.Threat, error) {
	s.log.Debug("creating threat", zap.String("title", input.Title))

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, svcerrors.ErrInvalidPayload
	}

	threatType := strings.TrimSpace(input.Type)
	if threatType == "" {
		return nil, svcerrors.ErrInvalidPayload
	}

	severity, ok := normalizeSeverity(input.Severity)
	if !ok {
		return nil, svcerrors.ErrInvalidPayload
	}

	indicator := strings.TrimSpace(input.Indicator)
	if indicator == "" {
		return nil, svcerrors.ErrInvalidPayload
	}

	if input.CreatedBy == uuid.Nil {
		return nil, svcerrors.ErrInvalidPayload
	}

	threat := &domain.Threat{
		Title:       title,
		Type:        threatType,
		Severity:    severity,
		Indicator:   indicator,
		Description: strings.TrimSpace(input.Description),
		CreatedBy:   input.CreatedBy,
	}

	if err := s.repo.Create(ctx, threat); err != nil {
		s.log.Error("repository Create failed", zap.Error(err))
		return nil, svcerrors.ErrInternal
	}

	return threat, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateThreatInput) (*domain.Threat, error) {
	threat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("repository GetByID failed", zap.Stringer("threat_id", id), zap.Error(err))
		return nil, err
	}
	if threat == nil {
		s.log.Info("threat not found", zap.Stringer("threat_id", id))
		return nil, svcerrors.ErrNotFound
	}

	if input.Title != nil {
		if trimmed := strings.TrimSpace(*input.Title); trimmed == "" {
			return nil, svcerrors.ErrInvalidPayload
		} else {
			threat.Title = trimmed
		}
	}

	if input.Type != nil {
		if trimmed := strings.TrimSpace(*input.Type); trimmed == "" {
			return nil, svcerrors.ErrInvalidPayload
		} else {
			threat.Type = trimmed
		}
	}

	if input.Severity != nil {
		severity, ok := normalizeSeverity(*input.Severity)
		if !ok {
			return nil, svcerrors.ErrInvalidPayload
		}
		threat.Severity = severity
	}

	if input.Indicator != nil {
		if trimmed := strings.TrimSpace(*input.Indicator); trimmed == "" {
			return nil, svcerrors.ErrInvalidPayload
		} else {
			threat.Indicator = trimmed
		}
	}

	if input.Description != nil {
		threat.Description = strings.TrimSpace(*input.Description)
	}

	if _, err := s.repo.Update(ctx, threat); err != nil {
		s.log.Error("repository Update failed", zap.Stringer("threat_id", id), zap.Error(err))
		return nil, svcerrors.ErrInternal
	}

	return threat, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.repo.Delete(ctx, id)
	if err != nil {
		s.log.Error("repository Delete failed", zap.Stringer("threat_id", id), zap.Error(err))
		return err
	}
	if !deleted {
		return svcerrors.ErrNotFound
	}
	return nil
}
