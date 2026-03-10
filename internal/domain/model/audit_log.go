package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog keeps an immutable record of user activity.
type AuditLog struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       *uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;index"`
	Action       string     `json:"action" gorm:"type:varchar(255);not null"`
	ResourceType string     `json:"resource_type" gorm:"type:varchar(100);not null"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty" gorm:"type:uuid"`
	IPAddress    string     `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent    string     `json:"user_agent" gorm:"type:varchar(512)"`
	Status       string     `json:"status" gorm:"type:varchar(20);not null"`
	Metadata     []byte     `json:"metadata" gorm:"type:jsonb"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
}
