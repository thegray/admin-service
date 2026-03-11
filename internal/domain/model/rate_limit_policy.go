package domain

import (
	"time"

	"github.com/google/uuid"
)

type RateLimitPolicy struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Scope             string     `json:"scope" gorm:"type:varchar(32);not null;index:idx_rate_limit_scope_role_resource,priority:1;uniqueIndex:uidx_rate_limit_scope_role_resource,priority:1"`
	RoleID            *uuid.UUID `json:"role_id,omitempty" gorm:"type:uuid;index:idx_rate_limit_scope_role_resource,priority:2;uniqueIndex:uidx_rate_limit_scope_role_resource,priority:2"`
	Resource          string     `json:"resource" gorm:"type:varchar(64);not null;index:idx_rate_limit_scope_role_resource,priority:3;uniqueIndex:uidx_rate_limit_scope_role_resource,priority:3"`
	RequestsPerMinute int        `json:"requests_per_minute" gorm:"not null"`
	Burst             int        `json:"burst" gorm:"not null"`
	CreatedAt         time.Time  `json:"created_at" gorm:"autoCreateTime"`
	Role              Role       `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}
