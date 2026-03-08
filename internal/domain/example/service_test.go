package example

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"admin-service/internal/domain"
	pkgerrors "admin-service/pkg/errors"

	"go.uber.org/zap"
)

type mockRepo struct {
	getByID  func(context.Context, int64) (*domain.Example, error)
	insert   func(context.Context, *domain.Example) error
	inserted *domain.Example
}

func (m *mockRepo) GetByID(ctx context.Context, id int64) (*domain.Example, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockRepo) Insert(ctx context.Context, ex *domain.Example) error {
	m.inserted = ex
	if m.insert != nil {
		return m.insert(ctx, ex)
	}
	return nil
}

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("repo failure")
	sample := &domain.Example{ID: 1, Message: "hello"}

	cases := []struct {
		name    string
		repo    Repository
		id      int64
		want    *domain.Example
		wantErr error
	}{
		{
			name: "found",
			repo: &mockRepo{
				getByID: func(context.Context, int64) (*domain.Example, error) {
					return sample, nil
				},
			},
			id:   1,
			want: sample,
		},
		{
			name: "not found",
			repo: &mockRepo{
				getByID: func(context.Context, int64) (*domain.Example, error) {
					return nil, nil
				},
			},
			id:      2,
			wantErr: pkgerrors.ErrNotFound,
		},
		{
			name: "repo error",
			repo: &mockRepo{
				getByID: func(context.Context, int64) (*domain.Example, error) {
					return nil, repoErr
				},
			},
			id:      3,
			wantErr: repoErr,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, zap.NewNop())
			got, err := svc.GetByID(ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetByID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetByID() unexpected error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetByID() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo, zap.NewNop())
		msg := "new message"

		if _, err := svc.Create(ctx, msg); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}

		if repo.inserted == nil {
			t.Fatal("Create() did not insert an example")
		}

		if repo.inserted.Message != msg {
			t.Fatalf("Create() inserted message = %q, want %q", repo.inserted.Message, msg)
		}
	})

	t.Run("repo failure", func(t *testing.T) {
		repoErr := errors.New("insert fail")
		repo := &mockRepo{
			insert: func(context.Context, *domain.Example) error {
				return repoErr
			},
		}
		svc := NewService(repo, zap.NewNop())

		if _, err := svc.Create(ctx, "doesn't matter"); !errors.Is(err, pkgerrors.ErrInternal) {
			t.Fatalf("Create() error = %v, want ErrInternal", err)
		}
	})
}
