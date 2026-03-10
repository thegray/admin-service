package postgres

import (
	domain "admin-service/internal/domain/model"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Role{},
		&domain.Permission{},
		&domain.RolePermission{},
		&domain.UserRole{},
		&domain.Threat{},
		&domain.AuditLog{},
	)
}
