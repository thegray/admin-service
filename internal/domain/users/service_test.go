package users

import (
	"context"
	"errors"
	"testing"

	domain "admin-service/internal/domain/model"
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

	var captured *domain.User
	repo := &mockRepo{
		createFn: func(ctx context.Context, user *domain.User) error {
			captured = user
			return nil
		},
	}

	svc := NewService(repo, nil, zap.NewNop())
	user, err := svc.Create(context.Background(), nil, input)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, captured)
	require.Equal(t, captured, user)
	require.Equal(t, input.Email, user.Email)
	require.NotEqual(t, input.Password, user.Password)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)))
}

func TestServiceCreateInvalidPayload(t *testing.T) {
	svc := NewService(&mockRepo{}, nil, zap.NewNop())
	_, err := svc.Create(context.Background(), nil, CreateUserInput{Email: "", Password: ""})
	require.ErrorIs(t, err, svcerrors.ErrInvalidPayload)
}

func TestServiceUpdateNotFound(t *testing.T) {
	repo := &mockRepo{
		getFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return nil, nil
		},
	}

	svc := NewService(repo, nil, zap.NewNop())
	_, err := svc.Update(context.Background(), nil, uuid.New(), UpdateUserInput{})
	require.ErrorIs(t, err, svcerrors.ErrNotFound)
}

func TestServiceDeleteDeletedMissing(t *testing.T) {
	repo := &mockRepo{
		softDeleteFn: func(ctx context.Context, id uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := NewService(repo, nil, zap.NewNop())
	err := svc.Delete(context.Background(), nil, uuid.New())
	require.ErrorIs(t, err, svcerrors.ErrNotFound)
}

func TestServiceListDefaults(t *testing.T) {
	var seenLimit, seenOffset int
	repo := &mockRepo{
		listFn: func(ctx context.Context, limit, offset int) ([]*domain.User, error) {
			seenLimit = limit
			seenOffset = offset
			return []*domain.User{{Email: "a@example.com"}}, nil
		},
	}

	svc := NewService(repo, nil, zap.NewNop())
	list, err := svc.List(context.Background(), 0, -1)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 100, seenLimit)
	require.Equal(t, 0, seenOffset)
}

type mockRepo struct {
	getFn                   func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	getByEmailFn            func(ctx context.Context, email string) (*domain.User, error)
	listFn                  func(ctx context.Context, limit, offset int) ([]*domain.User, error)
	createFn                func(ctx context.Context, user *domain.User) error
	updateFn                func(ctx context.Context, user *domain.User) (bool, error)
	softDeleteFn            func(ctx context.Context, id uuid.UUID) (bool, error)
	getRolesFn              func(ctx context.Context, id uuid.UUID) ([]string, error)
	getPermissionsFn        func(ctx context.Context, id uuid.UUID) ([]string, error)
	incrementTokenVersionFn func(ctx context.Context, id uuid.UUID) (int64, error)
	getRoleByIDFn           func(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	assignRoleFn            func(ctx context.Context, userID, roleID uuid.UUID) error
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if m.listFn != nil {
		return m.listFn(ctx, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) Create(ctx context.Context, user *domain.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return errors.New("not implemented")
}

func (m *mockRepo) Update(ctx context.Context, user *domain.User) (bool, error) {
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

func (m *mockRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) GetRoles(ctx context.Context, id uuid.UUID) ([]string, error) {
	if m.getRolesFn != nil {
		return m.getRolesFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) GetPermissions(ctx context.Context, id uuid.UUID) ([]string, error) {
	if m.getPermissionsFn != nil {
		return m.getPermissionsFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) IncrementTokenVersion(ctx context.Context, id uuid.UUID) (int64, error) {
	if m.incrementTokenVersionFn != nil {
		return m.incrementTokenVersionFn(ctx, id)
	}
	return 0, errors.New("not implemented")
}

func (m *mockRepo) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	if m.getRoleByIDFn != nil {
		return m.getRoleByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if m.assignRoleFn != nil {
		return m.assignRoleFn(ctx, userID, roleID)
	}
	return errors.New("not implemented")
}
