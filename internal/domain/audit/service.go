package audit

import (
	"context"
	"encoding/json"

	domain "admin-service/internal/domain/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	StatusSuccess      = "success"
	StatusFailure      = "failure"
	defaultWorkerCount = 4
	defaultQueueSize   = 256
)

type Repository interface {
	Insert(ctx context.Context, entry *domain.AuditLog) error
}

type Service struct {
	repo  Repository
	log   *zap.Logger
	queue chan *domain.AuditLog
}

type RecordInput struct {
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	Status       string
	IPAddress    string
	UserAgent    string
	Metadata     map[string]any
}

func NewService(repo Repository, log *zap.Logger) *Service {
	if repo == nil {
		return nil
	}
	if log == nil {
		log = zap.NewNop()
	}

	queue := make(chan *domain.AuditLog, defaultQueueSize)
	s := &Service{
		repo:  repo,
		log:   log.Named("audit-service"),
		queue: queue,
	}
	s.startWorkers()
	return s
}

func (s *Service) startWorkers() {
	for i := 0; i < defaultWorkerCount; i++ {
		go s.worker()
	}
}

func (s *Service) worker() {
	for entry := range s.queue {
		s.insert(entry)
	}
}

func (s *Service) insert(entry *domain.AuditLog) {
	if err := s.repo.Insert(context.Background(), entry); err != nil {
		s.log.Warn("failed to persist audit log", zap.Error(err))
	}
}

func (s *Service) enqueue(entry *domain.AuditLog) {
	select {
	case s.queue <- entry:
	default:
		s.log.Warn("audit queue full, writing synchronously", zap.String("action", entry.Action))
		s.insert(entry)
	}
}

func (s *Service) Record(ctx context.Context, input RecordInput) {
	if s == nil || s.repo == nil {
		return
	}

	status := input.Status
	if status == "" {
		status = StatusSuccess
	}

	data, err := json.Marshal(input.Metadata)
	if err != nil {
		s.log.Warn("failed to marshal metadata", zap.Error(err))
		data = []byte("null")
	}

	entry := &domain.AuditLog{
		UserID:       input.ActorID,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		IPAddress:    input.IPAddress,
		UserAgent:    input.UserAgent,
		Status:       status,
		Metadata:     data,
	}

	s.enqueue(entry)
}
