package initpkg

import (
	"context"
	"fmt"
	"strings"

	domain "admin-service/internal/domain/model"
	"admin-service/internal/domain/users"
	"admin-service/pkg/config"
	"admin-service/pkg/middleware"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	adminRoleName   = "admin"
	viewerRoleName  = "viewer"
	analystRoleName = "analyst"
)

const adminEmailKey = "ADMIN_EMAIL"
const adminPasswordKey = "ADMIN_PASSWORD"

var adminPermissions = []string{
	middleware.PermissionUsersRead,
	middleware.PermissionUsersWrite,
	middleware.PermissionUsersDelete,
	middleware.PermissionThreatsRead,
	middleware.PermissionThreatsWrite,
	middleware.PermissionThreatsDelete,
}

var viewerPermissions = []string{
	middleware.PermissionUsersRead,
}

var analystPermissions = []string{
	middleware.PermissionThreatsRead,
	middleware.PermissionThreatsWrite,
	middleware.PermissionThreatsDelete,
}

// InitAdmin resolves the necessary secrets and seeds the admin/viewer roles + optional admin user.
func InitAdmin(ctx context.Context, cfg config.Config, db *gorm.DB, repo users.Repository, svc *users.Service, log *zap.Logger) error {
	if db == nil {
		return fmt.Errorf("db is required for admin initialization")
	}
	adminEmail, hasAdminEmail, err := config.ResolveOptionalSecret(ctx, cfg.Environment, adminEmailKey)
	if err != nil {
		return fmt.Errorf("resolve admin email: %w", err)
	}

	var adminPassword string
	if hasAdminEmail {
		adminPassword, err = config.ResolveSecret(ctx, cfg.Environment, adminPasswordKey)
		if err != nil {
			return fmt.Errorf("resolve admin password: %w", err)
		}
	}

	return seedAdmin(ctx, db, repo, svc, log, adminEmail, adminPassword)
}

func seedAdmin(ctx context.Context, db *gorm.DB, repo users.Repository, svc *users.Service, log *zap.Logger, email, password string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if db == nil || repo == nil || svc == nil {
		return fmt.Errorf("missing dependency for admin seeding")
	}
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)

	tx := db.WithContext(ctx)
	permIDs := make(map[string]uuid.UUID, len(adminPermissions))
	for _, name := range adminPermissions {
		perm := &domain.Permission{}
		if err := tx.FirstOrCreate(perm, domain.Permission{Name: name}).Error; err != nil {
			return fmt.Errorf("ensuring permission %s: %w", name, err)
		}
		permIDs[name] = perm.ID
	}

	adminRole := &domain.Role{}
	if err := tx.FirstOrCreate(adminRole, domain.Role{Name: adminRoleName}).Error; err != nil {
		return fmt.Errorf("ensuring admin role: %w", err)
	}
	if err := linkRolePermissions(tx, adminRole.ID, permIDs, adminPermissions); err != nil {
		return err
	}

	viewerRole := &domain.Role{}
	if err := tx.FirstOrCreate(viewerRole, domain.Role{Name: viewerRoleName}).Error; err != nil {
		return fmt.Errorf("ensuring viewer role: %w", err)
	}
	if err := linkRolePermissions(tx, viewerRole.ID, permIDs, viewerPermissions); err != nil {
		return err
	}

	analystRole := &domain.Role{}
	if err := tx.FirstOrCreate(analystRole, domain.Role{Name: analystRoleName}).Error; err != nil {
		return fmt.Errorf("ensuring analyst role: %w", err)
	}
	if err := linkRolePermissions(tx, analystRole.ID, permIDs, analystPermissions); err != nil {
		return err
	}

	if email == "" {
		if log != nil {
			log.Info("admin email not configured; seeded roles/permissions only")
		}
		return nil
	}
	if password == "" {
		return fmt.Errorf("admin password must be set when admin email is provided")
	}

	user, err := repo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("looking up admin user: %w", err)
	}
	if user == nil {
		created, err := svc.Create(ctx, users.CreateUserInput{
			Email:    email,
			Password: password,
			IsActive: true,
			RoleID:   &adminRole.ID,
		})
		if err != nil {
			return fmt.Errorf("creating admin user: %w", err)
		}
		user = created
	} else if !user.IsActive {
		user.IsActive = true
		if _, err := repo.Update(ctx, user); err != nil {
			return fmt.Errorf("activating admin user: %w", err)
		}
	}

	userRole := &domain.UserRole{UserID: user.ID, RoleID: adminRole.ID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(userRole).Error; err != nil {
		return fmt.Errorf("assigning admin role: %w", err)
	}

	if log != nil {
		log.Info("admin user seeded", zap.String("email", email), zap.String("role", adminRoleName))
	}
	return nil
}

func linkRolePermissions(tx *gorm.DB, roleID uuid.UUID, permIDs map[string]uuid.UUID, names []string) error {
	for _, name := range names {
		permID, ok := permIDs[name]
		if !ok {
			return fmt.Errorf("missing permission %s", name)
		}
		rp := &domain.RolePermission{RoleID: roleID, PermissionID: permID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(rp).Error; err != nil {
			return fmt.Errorf("linking role to permission: %w", err)
		}
	}
	return nil
}
