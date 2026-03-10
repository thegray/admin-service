package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email        string     `json:"email" gorm:"type:varchar(255);not null;uniqueIndex"`
	Password     string     `json:"-" gorm:"type:varchar(255);not null"`
	IsActive     bool       `json:"is_active" gorm:"not null;default:true"`
	TokenVersion int64      `json:"token_version" gorm:"not null;default:0"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}
