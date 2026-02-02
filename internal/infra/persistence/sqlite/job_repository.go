package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(ctx context.Context, j *job.Job) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO jobs (
			id, user_id, type, status, progress, input_payload,
			output_manifest, error_message, created_at, started_at,
			finished_at, expired_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		j.ID,
		j.UserID,
		j.Type,
		j.Status,
		j.Progress,
		j.InputPayload,
		j.OutputManifest,
		j.ErrorMessage,
		j.CreatedAt,
		j.StartedAt,
		j.FinishedAt,
		j.ExpiredAt,
	)
	return err
}

func (r *JobRepository) List(ctx context.Context) ([]*job.Job, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			user_id,
			type,
			status,
			progress,
			input_payload,
			output_manifest,
			error_message,
			created_at,
			started_at,
			finished_at,
			expired_at
		FROM jobs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*job.Job

	for rows.Next() {
		var j job.Job

		var outputManifest []byte
		var userID sql.NullString
		var errorMessage sql.NullString
		var startedAt sql.NullTime
		var finishedAt sql.NullTime
		var expiredAt sql.NullTime

		err := rows.Scan(
			&j.ID,
			&userID,
			&j.Type,
			&j.Status,
			&j.Progress,
			&j.InputPayload,
			&outputManifest,
			&errorMessage,
			&j.CreatedAt,
			&startedAt,
			&finishedAt,
			&expiredAt,
		)
		if err != nil {
			return nil, err
		}

		if userID.Valid {
			j.UserID = &userID.String
		}

		if len(outputManifest) > 0 {
			var manifest job.OutputManifest
			if err := json.Unmarshal(outputManifest, &manifest); err != nil {
				return nil, err
			}
			j.OutputManifest = &manifest
		}

		if errorMessage.Valid {
			j.ErrorMessage = &errorMessage.String
		}

		if startedAt.Valid {
			j.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			j.FinishedAt = &finishedAt.Time
		}
		if expiredAt.Valid {
			j.ExpiredAt = &expiredAt.Time
		}

		jobs = append(jobs, &j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *JobRepository) FindByID(ctx context.Context, id string) (*job.Job, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			user_id,
			type,
			status,
			progress,
			input_payload,
			output_manifest,
			error_message,
			created_at,
			started_at,
			finished_at,
			expired_at
		FROM jobs
		WHERE id = ?
	`, id)

	var j job.Job

	var (
		userID         sql.NullString
		errorMessage   sql.NullString
		startedAt      sql.NullTime
		finishedAt     sql.NullTime
		expiredAt      sql.NullTime
		outputManifest []byte
	)

	if err := row.Scan(
		&j.ID,
		&userID,
		&j.Type,
		&j.Status,
		&j.Progress,
		&j.InputPayload,
		&outputManifest,
		&errorMessage,
		&j.CreatedAt,
		&startedAt,
		&finishedAt,
		&expiredAt,
	); err != nil {
		return nil, err
	}

	if userID.Valid {
		j.UserID = &userID.String
	}

	if len(outputManifest) > 0 {
		var manifest job.OutputManifest
		if err := json.Unmarshal(outputManifest, &manifest); err != nil {
			return nil, err
		}
		j.OutputManifest = &manifest
	}

	if errorMessage.Valid {
		j.ErrorMessage = &errorMessage.String
	}

	if startedAt.Valid {
		j.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		j.FinishedAt = &finishedAt.Time
	}
	if expiredAt.Valid {
		j.ExpiredAt = &expiredAt.Time
	}

	return &j, nil
}

func (r *JobRepository) Update(ctx context.Context, j *job.Job) error {
	var outputManifest []byte
	var err error

	if j.OutputManifest != nil {
		outputManifest, err = json.Marshal(j.OutputManifest)
		if err != nil {
			return err
		}
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE jobs SET
			user_id = ?, type = ?, status = ?, progress = ?,
			input_payload = ?, output_manifest = ?, error_message = ?,
			started_at = ?, finished_at = ?, expired_at = ?
		WHERE id = ?
	`,
		j.UserID,
		j.Type,
		j.Status,
		j.Progress,
		j.InputPayload,
		outputManifest,
		j.ErrorMessage,
		j.StartedAt,
		j.FinishedAt,
		j.ExpiredAt,
		j.ID,
	)

	return err
}

func (r *JobRepository) UpdateStatus(ctx context.Context, id string, status job.JobStatus) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE jobs SET status = ? WHERE id = ?
	`, status, id)
	return err
}

func (r *JobRepository) UpdateProgress(ctx context.Context, id string, progress int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE jobs SET progress = ? WHERE id = ?
	`, progress, id)
	return err
}

func (r *JobRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM jobs WHERE id = ?", id)
	return err
}
