package submissionRepo

import (
	"context"
	"errors"

	"github.com/arabkood/backend/internal/domains/module/interfaces/exercise"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/jackc/pgx/v5"
)

type SubmissionRepository struct {
	querier postgres.Querier
}

func NewSubmissionRepository(querier postgres.Querier) *SubmissionRepository {
	return &SubmissionRepository{querier: querier}
}

func (r *SubmissionRepository) GetSubmissionByUserModule(ctx context.Context, userId, moduleId string) (*exercise.Submission, *appErrors.Error) {
	submission := &exercise.Submission{}

	query := `
        SELECT 
            user_id, module_id, 
            xp_reward, attempts, 
            created_at, updated_at,
            user_files, args, results
        FROM users.modules_submission
        WHERE user_id = $1 AND module_id = $2`

	err := r.querier.QueryRow(ctx, query, userId, moduleId).Scan(
		&submission.UserID, &submission.ModuleID,
		&submission.XpReward, &submission.Attempts,
		&submission.CreatedAt, &submission.UpdatedAt,
		&submission.UserFiles, &submission.Args, &submission.Results,
	)
	//

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return submission, nil
}
