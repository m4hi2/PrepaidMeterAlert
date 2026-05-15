package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/uptrace/bun"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByPlatformID(ctx context.Context, platform models.Platform, platformID string) (*models.User, error)
}

type userRepo struct {
	db *bun.DB
}

func NewUserRepo(db *bun.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.Must(uuid.NewV7())
	_, err := r.db.NewInsert().Model(user).Exec(ctx)
	return err
}

func (r *userRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().Model((*models.User)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := r.db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx)
	return user, err
}

func (r *userRepo) GetByPlatformID(ctx context.Context, platform models.Platform, platformID string) (*models.User, error) {
	user := &models.User{}
	err := r.db.NewSelect().Model(user).
		Where("platform = ? AND platform_id = ?", platform, platformID).
		Scan(ctx)
	return user, err
}
