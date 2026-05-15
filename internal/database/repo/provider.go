package repo

import (
	"context"

	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/uptrace/bun"
)

type ProviderRepository interface {
	GetActive(ctx context.Context) ([]models.ProviderCode, error)
}

type providerRepo struct {
	db *bun.DB
}

func NewProviderRepo(db *bun.DB) ProviderRepository {
	return &providerRepo{db: db}
}

func (r *providerRepo) GetActive(ctx context.Context) ([]models.ProviderCode, error) {
	var codes []models.ProviderCode
	err := r.db.NewSelect().
		TableExpr("providers").
		ColumnExpr("code").
		Where("enabled = true AND deleted_at IS NULL").
		Scan(ctx, &codes)
	return codes, err
}
