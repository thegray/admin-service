package users

import (
	"context"
	"errors"
	"testing"

	svcerrors "admin-service/pkg/errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func TestServiceCreateHashesPassword(t *testing.T) {
	input := CreateUserInput{
		Email:    "jane@example.com",
		Password: "secret-password",
		IsActive: true,
	}

	var captured *User
	repo := &mockRepo{
		createFn: func(ctx context.Context, user *User) error {
			captured = user
			return nil
		},
	}

	svc := NewService(repo, zap.NewNop())
	user, err := svc.Create(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, captured)
	require.Equal(t, captured, user)
	require.Equal(t, input.Email, user.Email)
	require.NotEqual(t, input.Password, user.Password)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)))
}

func TestServiceCreateInvalidPayload(t *testing.T) {
	svc := NewService(&mockRepo{}, zap.NewNop())
	_, err := svc.Create(context.Background(), CreateUserInput{Email: "", Password: ""})
	require.ErrorIs(t, err, svcerrors.ErrInvalidPayload)
}

func TestServiceUpdateNotFound(t *testing.T) {
	repo := &mockRepo{
		getFn: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return nil, nil
		},
	}

	svc := NewService(repo, zap.NewNop())
	_, err := svc.Update(context.Background(), uuid.New(), UpdateUserInput{})
	require.ErrorIs(t, err, svcerrors.ErrNotFound)
}

func TestServiceDeleteDeletedMissing(t *testing.T) {
	repo := &mockRepo{
		softDeleteFn: func(ctx context.Context, id uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := NewService(repo, zap.NewNop())
	err := svc.Delete(context.Background(), uuid.New())
	require.ErrorIs(t, err, svcerrors.ErrNotFound)
}

func TestServiceListDefaults(t *testing.T) {
	var seenLimit, seenOffset int
	repo := &mockRepo{
		listFn: func(ctx context.Context, limit, offset int) ([]*User, error) {
			seenLimit = limit
			seenOffset = offset
			return []*User{{Email: "a@example.com"}}, nil
		},
	}

	svc := NewService(repo, zap.NewNop())
	list, err := svc.List(context.Background(), 0, -1)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 100, seenLimit)
	require.Equal(t, 0, seenOffset)
}

type mockRepo struct {
	getFn        func(ctx context.Context, id uuid.UUID) (*User, error)
	listFn       func(ctx context.Context, limit, offset int) ([]*User, error)
	createFn     func(ctx context.Context, user *User) error
	updateFn     func(ctx context.Context, user *User) (bool, error)
	softDeleteFn func(ctx context.Context, id uuid.UUID) (bool, error)
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) List(ctx context.Context, limit, offset int) ([]*User, error) {
	if m.listFn != nil {
		return m.listFn(ctx, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) Create(ctx context.Context, user *User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return errors.New("not implemented")
}

func (m *mockRepo) Update(ctx context.Context, user *User) (bool, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return false, errors.New("not implemented")
}

func (m *mockRepo) SoftDelete(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return false, errors.New("not implemented")
}
