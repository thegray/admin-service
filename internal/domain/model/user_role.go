package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole struct {
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;primaryKey"`
	RoleID     uuid.UUID  `json:"role_id" gorm:"type:uuid;not null;primaryKey"`
	AssignedAt time.Time  `json:"assigned_at" gorm:"autoCreateTime"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty" gorm:"type:uuid"`
}
