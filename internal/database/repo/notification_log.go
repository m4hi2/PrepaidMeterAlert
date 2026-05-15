package repo

import (
	"context"

	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/uptrace/bun"
)

type NotificationLogRepository interface {
	Insert(ctx context.Context, log *models.NotificationLog) error
}

type notificationLogRepo struct {
	db *bun.DB
}

func NewNotificationLogRepo(db *bun.DB) NotificationLogRepository {
	return &notificationLogRepo{db: db}
}

func (r *notificationLogRepo) Insert(ctx context.Context, log *models.NotificationLog) error {
	_, err := r.db.NewInsert().Model(log).Exec(ctx)
	return err
}
