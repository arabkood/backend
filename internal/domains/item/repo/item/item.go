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

func (r *ItemRepository) GetItemById(ctx context.Context, id string) (*item.Item, *appErrors.Error) {
	rows, _ := r.querier.Query(ctx, "SELECT * FROM class.items WHERE id = $1 LIMIT 1", id)
	it, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[item.Item])

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return it, nil
}
