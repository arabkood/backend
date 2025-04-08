package repo

import (
	"context"
	"encoding/json"
	"errors"

	domainTrack "github.com/arabkood/backend/internal/domains/track/interfaces/track"
	"github.com/arabkood/backend/internal/postgres"
	appErrors "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TrackRepository struct {
	querier postgres.Querier
}

func NewTrackRepository(querier postgres.Querier) *TrackRepository {
	return &TrackRepository{querier: querier}
}

func (r *TrackRepository) GetAllTracks(ctx context.Context) ([]domainTrack.Track, *appErrors.Error) {
	var tracks []domainTrack.Track

	query := `
        SELECT 
            id, title, slug, description, logo, 
            programming_languages, difficulty, premium_only, 
            skills, tags, total_xp, total_modules, 
            estimated_hours, students, created_at, updated_at, 
            outcomes, requirements
        FROM class.tracks`

	rows, err := r.querier.Query(ctx, query)
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var track domainTrack.Track
		err := rows.Scan(
			&track.ID, &track.Title, &track.Slug, &track.Description, &track.Logo,
			&track.ProgrammingLanguages, &track.Difficulty, &track.PremiumOnly,
			&track.Skills, &track.Tags, &track.TotalXP, &track.TotalModules,
			&track.EstimatedHours, &track.Students, &track.CreatedAt, &track.UpdatedAt,
			&track.Outcomes, &track.Requirements,
		)
		if err != nil {
			return nil, appErrors.ErrorInternal().WithError(err)
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}

func (r *TrackRepository) GetTrackByID(ctx context.Context, id uuid.UUID) (*domainTrack.Track, *appErrors.Error) {
	var track domainTrack.Track

	query := `
        SELECT 
            id, title, slug, description, logo, 
            programming_languages, difficulty, premium_only, 
            skills, tags, total_xp, total_modules, 
            estimated_hours, students, created_at, updated_at, 
            outcomes, requirements
        FROM class.tracks
        WHERE id = $1`

	err := r.querier.QueryRow(ctx, query, id).Scan(
		&track.ID, &track.Title, &track.Slug, &track.Description, &track.Logo,
		&track.ProgrammingLanguages, &track.Difficulty, &track.PremiumOnly,
		&track.Skills, &track.Tags, &track.TotalXP, &track.TotalModules,
		&track.EstimatedHours, &track.Students, &track.CreatedAt, &track.UpdatedAt,
		&track.Outcomes, &track.Requirements,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return &track, nil
}

func (r *TrackRepository) GetTrackBySlug(ctx context.Context, slug string) (*domainTrack.Track, *appErrors.Error) {
	var track domainTrack.Track

	query := `
        SELECT 
            id, title, slug, description, logo, 
            programming_languages, difficulty, premium_only, 
            skills, tags, total_xp, total_modules, 
            estimated_hours, students, created_at, updated_at, 
            outcomes, requirements
        FROM class.tracks
        WHERE slug = $1`

	err := r.querier.QueryRow(ctx, query, slug).Scan(
		&track.ID, &track.Title, &track.Slug, &track.Description, &track.Logo,
		&track.ProgrammingLanguages, &track.Difficulty, &track.PremiumOnly,
		&track.Skills, &track.Tags, &track.TotalXP, &track.TotalModules,
		&track.EstimatedHours, &track.Students, &track.CreatedAt, &track.UpdatedAt,
		&track.Outcomes, &track.Requirements,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrorNotFound()
	}
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	return &track, nil
}

func (r *TrackRepository) GetTrackContentByTrackID(ctx context.Context, trackID uuid.UUID) ([]domainTrack.SectionWithModules, *appErrors.Error) {
	query := `
		WITH track_sections AS (
			SELECT
				s.id,
				s.track_id,
				s.title,
				s.description,
				s.order_number,
				s.created_at,
				s.updated_at,
				COALESCE(json_agg(m ORDER BY m.order_number) FILTER (WHERE m.id IS NOT NULL), '[]') AS modules
			FROM
				class.tracks_sections s
			LEFT JOIN
				class.modules m ON s.id = m.section_id
			WHERE s.track_id = $1
			GROUP BY
				s.id, s.track_id, s.title, s.description, s.order_number, s.created_at, s.updated_at
		)
		SELECT
			COALESCE(json_agg(ts ORDER BY ts.order_number), '[]') AS sections
		FROM
			track_sections ts;
	`

	var sectionsJSON string
	err := r.querier.QueryRow(ctx, query, trackID).Scan(&sectionsJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// It's possible a track has no sections/modules, so this isn't an error
			return []domainTrack.SectionWithModules{}, nil
		}
		return nil, appErrors.ErrorInternal().WithError(err)
	}

	var sections []domainTrack.SectionWithModules
	err = json.Unmarshal([]byte(sectionsJSON), &sections)
	if err != nil {
		return nil, appErrors.ErrorInternal().WithError(err).WithMessage("Error unmarshalling sections")
	}

	return sections, nil
}

func (r *TrackRepository) StartTrack(ctx context.Context, userID, trackID uuid.UUID) *appErrors.Error {
	query := `
        INSERT INTO users.track (
            user_id, track_id, started_at
        ) VALUES (
            $1, $2, NOW()
        )`

	_, err := r.querier.Exec(ctx, query, userID, trackID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Check for unique violation error code
			if pgErr.ConstraintName == "user_tracks_pkey" {
				return appErrors.ErrorTrackAlreadyStarted()
			}
		}
		return appErrors.ErrorInternal().WithError(err)
	}

	return nil
}
