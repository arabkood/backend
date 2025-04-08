package repo

import (
	"context"
	"errors"

	"github.com/arabkood/backend/internal/domains/module/interfaces/module"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/jackc/pgx/v5"
)

type ModuleRepository struct {
	querier postgres.Querier
}

func NewModuleRepository(querier postgres.Querier) *ModuleRepository {
	return &ModuleRepository{querier: querier}
}

func (r *ModuleRepository) GetModuleBySlug(ctx context.Context, slug string) (*module.Module, *appErrors.Error) {
	module := &module.Module{}

	query := `
        SELECT 
            id, slug, track_id, title, description, order_number, 
            created_at, updated_at, xp_reward, estimated_minutes, 
            difficulty, dependencies, premium_only, section_id, 
            type, source, deleted_at, pre_args
        FROM class.modules
        WHERE slug = $1 AND deleted_at is NULL`

	err := r.querier.QueryRow(ctx, query, slug).Scan(
		&module.ID, &module.Slug, &module.TrackID, &module.Title, &module.Description,
		&module.OrderNumber, &module.CreatedAt, &module.UpdatedAt, &module.XPReward,
		&module.EstimatedMinutes, &module.Difficulty, &module.Dependencies,
		&module.PremiumOnly, &module.SectionID, &module.Type, &module.Source,
		&module.DeletedAt, &module.PreArgs,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return module, nil
}

func (r *ModuleRepository) GetModuleById(ctx context.Context, id string) (*module.Module, *appErrors.Error) {
	module := &module.Module{}

	query := `
        SELECT 
            id, slug, track_id, title, description, order_number, 
            created_at, updated_at, xp_reward, estimated_minutes, 
            difficulty, dependencies, premium_only, section_id, 
            type, source, deleted_at, pre_args
        FROM class.modules
        WHERE id = $1 AND deleted_at is NULL`

	err := r.querier.QueryRow(ctx, query, id).Scan(
		&module.ID, &module.Slug, &module.TrackID, &module.Title, &module.Description,
		&module.OrderNumber, &module.CreatedAt, &module.UpdatedAt, &module.XPReward,
		&module.EstimatedMinutes, &module.Difficulty, &module.Dependencies,
		&module.PremiumOnly, &module.SectionID, &module.Type, &module.Source,
		&module.DeletedAt, &module.PreArgs,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return module, nil
}
