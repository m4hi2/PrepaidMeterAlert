package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/uptrace/bun"
)

type MeterRepository interface {
	Create(ctx context.Context, meter *models.Meter) error
	Update(ctx context.Context, meter *models.Meter) error
	GetAll(ctx context.Context) ([]*models.Meter, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Meter, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Meter, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type meterRepo struct {
	db *bun.DB
}

func NewMeterRepo(db *bun.DB) MeterRepository {
	return &meterRepo{db: db}
}

func (r *meterRepo) Create(ctx context.Context, meter *models.Meter) error {
	meter.ID = uuid.Must(uuid.NewV7())
	_, err := r.db.NewInsert().Model(meter).Exec(ctx)
	return err
}

// Update saves the full meter struct to the DB. The caller is responsible for
// loading the meter first and mutating only the intended fields.
func (r *meterRepo) Update(ctx context.Context, meter *models.Meter) error {
	_, err := r.db.NewUpdate().Model(meter).WherePK().Exec(ctx)
	return err
}

func (r *meterRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Meter, error) {
	meter := &models.Meter{}
	err := r.db.NewSelect().Model(meter).Where("id = ?", id).Scan(ctx)
	return meter, err
}

func (r *meterRepo) GetAll(ctx context.Context) ([]*models.Meter, error) {
	var meters []*models.Meter
	err := r.db.NewSelect().Model(&meters).Scan(ctx)
	return meters, err
}

func (r *meterRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Meter, error) {
	var meters []*models.Meter
	err := r.db.NewSelect().Model(&meters).Where("user_id = ?", userID).Scan(ctx)
	return meters, err
}

func (r *meterRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().Model((*models.Meter)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}
