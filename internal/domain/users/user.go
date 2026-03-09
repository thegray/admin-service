package users

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email     string     `json:"email" gorm:"type:varchar(255);not null;uniqueIndex"`
	Password  string     `json:"-" gorm:"type:varchar(255);not null"`
	IsActive  bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	List(ctx context.Context, limit, offset int) ([]*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) (bool, error)
	SoftDelete(ctx context.Context, id uuid.UUID) (bool, error)
}
