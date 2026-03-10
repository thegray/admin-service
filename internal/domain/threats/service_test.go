package threats

import (
	"context"
	"errors"
	"testing"

	domain "admin-service/internal/domain/model"
	svcerrors "admin-service/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockThreatRepo struct {
	list    func(context.Context, int, int) ([]*domain.Threat, error)
	getByID func(context.Context, uuid.UUID) (*domain.Threat, error)
	create  func(context.Context, *domain.Threat) error
	update  func(context.Context, *domain.Threat) (bool, error)
	delete  func(context.Context, uuid.UUID) (bool, error)
}

func (m *mockThreatRepo) List(ctx context.Context, limit, offset int) ([]*domain.Threat, error) {
	if m.list != nil {
		return m.list(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockThreatRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Threat, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockThreatRepo) Create(ctx context.Context, threat *domain.Threat) error {
	if m.create != nil {
		return m.create(ctx, threat)
	}
	return nil
}

func (m *mockThreatRepo) Update(ctx context.Context, threat *domain.Threat) (bool, error) {
	if m.update != nil {
		return m.update(ctx, threat)
	}
	return false, nil
}

func (m *mockThreatRepo) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return false, nil
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	sample := &domain.Threat{ID: uuid.New()}
	var seenLimit, seenOffset int
	repo := &mockThreatRepo{
		list: func(ctx context.Context, limit, offset int) ([]*domain.Threat, error) {
			seenLimit = limit
			seenOffset = offset
			return []*domain.Threat{sample}, nil
		},
	}

	svc := NewService(repo, zap.NewNop())

	list, err := svc.List(ctx, 0, -5)
	if err != nil {
		t.Fatalf("List() unexpected error = %v", err)
	}

	if len(list) != 1 || list[0] != sample {
		t.Fatalf("List() = %v, want %v", list, []*domain.Threat{sample})
	}

	if seenLimit != 100 || seenOffset != 0 {
		t.Fatalf("List() default pagination = (%d, %d), want (100, 0)", seenLimit, seenOffset)
	}
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	input := CreateThreatInput{
		Title:       "  Title ",
		Type:        " Malware ",
		Severity:    "HIGH",
		Indicator:   " 1.1.1.1 ",
		Description: "  desc  ",
		CreatedBy:   uuid.New(),
	}
	var stored *domain.Threat
	repo := &mockThreatRepo{
		create: func(_ context.Context, threat *domain.Threat) error {
			stored = threat
			return nil
		},
	}
	svc := NewService(repo, zap.NewNop())

	got, err := svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create() unexpected error = %v", err)
	}

	if got.Severity != "high" {
		t.Fatalf("Create() severity = %q, want %q", got.Severity, "high")
	}

	if stored == nil || stored.Title != "Title" {
		t.Fatalf("Create() stored title = %q, want %q", stored.Title, "Title")
	}

	if stored.Type != "Malware" {
		t.Fatalf("Create() stored type = %q, want %q", stored.Type, "Malware")
	}

	if stored.Indicator != "1.1.1.1" {
		t.Fatalf("Create() stored indicator = %q, want %q", stored.Indicator, "1.1.1.1")
	}

	if stored.Description != "desc" {
		t.Fatalf("Create() stored description = %q, want %q", stored.Description, "desc")
	}

	if got.CreatedBy != input.CreatedBy {
		t.Fatalf("Create() created_by = %v, want %v", got.CreatedBy, input.CreatedBy)
	}

	_, err = svc.Create(ctx, CreateThreatInput{Title: "a", Type: "b", Severity: "invalid", Indicator: "c", CreatedBy: uuid.New()})
	if !errors.Is(err, svcerrors.ErrInvalidPayload) {
		t.Fatalf("Create() severity validation error = %v, want ErrInvalidPayload", err)
	}
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	existing := &domain.Threat{
		ID:        id,
		Title:     "old",
		Type:      "oldtype",
		Severity:  "low",
		Indicator: "old",
	}
	repo := &mockThreatRepo{
		getByID: func(context.Context, uuid.UUID) (*domain.Threat, error) {
			return existing, nil
		},
		update: func(_ context.Context, threat *domain.Threat) (bool, error) {
			return true, nil
		},
	}
	svc := NewService(repo, zap.NewNop())

	newSeverity := "CRITICAL"
	resp, err := svc.Update(ctx, id, UpdateThreatInput{
		Title:    ptr(" new "),
		Type:     ptr("type"),
		Severity: &newSeverity,
	})
	if err != nil {
		t.Fatalf("Update() unexpected error = %v", err)
	}

	if resp.Title != "new" || resp.Severity != "critical" {
		t.Fatalf("Update() result = %+v", resp)
	}

	_, err = svc.Update(ctx, id, UpdateThreatInput{Severity: ptr("wrong")})
	if !errors.Is(err, svcerrors.ErrInvalidPayload) {
		t.Fatalf("Update() bad severity = %v, want ErrInvalidPayload", err)
	}

	repo.getByID = func(context.Context, uuid.UUID) (*domain.Threat, error) { return nil, nil }
	_, err = svc.Update(ctx, id, UpdateThreatInput{Title: ptr("ok")})
	if !errors.Is(err, svcerrors.ErrNotFound) {
		t.Fatalf("Update() not found = %v, want ErrNotFound", err)
	}
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	repo := &mockThreatRepo{
		delete: func(context.Context, uuid.UUID) (bool, error) { return true, nil },
	}
	svc := NewService(repo, zap.NewNop())

	if err := svc.Delete(ctx, id); err != nil {
		t.Fatalf("Delete() unexpected error = %v", err)
	}

	repo.delete = func(context.Context, uuid.UUID) (bool, error) { return false, nil }
	if err := svc.Delete(ctx, id); !errors.Is(err, svcerrors.ErrNotFound) {
		t.Fatalf("Delete() not found = %v, want ErrNotFound", err)
	}
}

func ptr(s string) *string { return &s }
