package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/uptrace/bun"
)

type FeedbackRepository interface {
	Create(ctx context.Context, feedback *models.Feedback) error
}

type feedbackRepo struct {
	db *bun.DB
}

func NewFeedbackRepo(db *bun.DB) FeedbackRepository {
	return &feedbackRepo{db: db}
}

func (r *feedbackRepo) Create(ctx context.Context, feedback *models.Feedback) error {
	feedback.ID = uuid.Must(uuid.NewV7())
	_, err := r.db.NewInsert().Model(feedback).Exec(ctx)
	return err
}
