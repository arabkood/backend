package repoItem

import (
	"context"
	"errors"

	"github.com/arabkood/backend/internal/domains/item/interfaces/item"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/jackc/pgx/v5"
)

type ItemRepository struct {
	querier postgres.Querier
}

func NewItemRepository(querier postgres.Querier) *ItemRepository {
	return &ItemRepository{querier: querier}
}

func (r *ItemRepository) GetItemBySlug(ctx context.Context, slug string) (*item.Item, *appErrors.Error) {
	item := &item.Item{}

	err := r.querier.QueryRow(ctx, "SELECT * FROM class.items WHERE slug = $1", slug).Scan(item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return item, nil
}

func (r *ItemRepository) GetItemById(ctx context.Context, id string) (*item.Item, *appErrors.Error) {
	item := &item.Item{}

	err := r.querier.QueryRow(ctx, "SELECT * FROM class.items WHERE id = $1", id).Scan(item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return item, nil
}
