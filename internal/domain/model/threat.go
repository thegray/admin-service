package domain

import (
	"time"

	"github.com/google/uuid"
)

// Threat represents a tracked security threat indicator.
type Threat struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Title       string    `json:"title" gorm:"type:varchar(255);not null"`
	Type        string    `json:"type" gorm:"type:varchar(100);not null"`
	Severity    string    `json:"severity" gorm:"type:varchar(20);not null"`
	Indicator   string    `json:"indicator" gorm:"type:varchar(255);not null"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedBy   uuid.UUID `json:"created_by" gorm:"type:uuid;not null;index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
