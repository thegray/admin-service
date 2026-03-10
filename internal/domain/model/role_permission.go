package domain

import (
	"github.com/google/uuid"
)

type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id" gorm:"type:uuid;not null;primaryKey"`
	PermissionID uuid.UUID `json:"permission_id" gorm:"type:uuid;not null;primaryKey"`
}
