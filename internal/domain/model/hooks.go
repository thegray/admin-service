package domain

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ensureUUIDv7(id *uuid.UUID) error {
	if id == nil || *id != uuid.Nil {
		return nil
	}
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	*id = newID
	return nil
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	return ensureUUIDv7(&u.ID)
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	return ensureUUIDv7(&r.ID)
}

func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	return ensureUUIDv7(&p.ID)
}
