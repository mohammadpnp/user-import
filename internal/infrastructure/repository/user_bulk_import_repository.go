package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/mohammadpnp/user-import/internal/domain/user"
)

const uuidRegex = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$"

type UserBulkImportRepository struct {
	pool *pgxpool.Pool
}

func NewUserBulkImportRepository(pool *pgxpool.Pool) *UserBulkImportRepository {
	return &UserBulkImportRepository{pool: pool}
}

func (r *UserBulkImportRepository) ImportChunk(ctx context.Context, jobID string, users []domain.User) (domain.ImportChunkResult, error) {
	if len(users) == 0 {
		return domain.ImportChunkResult{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.ImportChunkResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	userRows := make([][]any, 0, len(users))
	addressRows := make([][]any, 0)
	for i, user := range users {
		userRows = append(userRows, []any{jobID, int64(i), nullableText(user.ID), user.Name, user.Email, user.PhoneNumber})
		for _, address := range user.Addresses {
			addressRows = append(addressRows, []any{
				jobID,
				int64(i),
				nullableText(user.ID),
				user.Email,
				address.Street,
				address.City,
				address.State,
				address.ZipCode,
				address.Country,
			})
		}
	}

	if _, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"stg_users"},
		[]string{"job_id", "row_index", "external_id", "name", "email", "phone_number"},
		pgx.CopyFromRows(userRows),
	); err != nil {
		return domain.ImportChunkResult{}, fmt.Errorf("copy users staging: %w", err)
	}

	if len(addressRows) > 0 {
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"stg_addresses"},
			[]string{"job_id", "row_index", "user_external_id", "user_email", "street", "city", "state", "zip_code", "country"},
			pgx.CopyFromRows(addressRows),
		); err != nil {
			return domain.ImportChunkResult{}, fmt.Errorf("copy addresses staging: %w", err)
		}
	}

	imported, updated, err := upsertUsersByExternalID(ctx, tx, jobID)
	if err != nil {
		return domain.ImportChunkResult{}, err
	}

	importedByEmail, updatedByEmail, err := upsertUsersByEmail(ctx, tx, jobID)
	if err != nil {
		return domain.ImportChunkResult{}, err
	}
	imported += importedByEmail
	updated += updatedByEmail

	if err := replaceAddresses(ctx, tx, jobID); err != nil {
		return domain.ImportChunkResult{}, err
	}

	if _, err := tx.Exec(ctx, "DELETE FROM stg_addresses WHERE job_id = $1", jobID); err != nil {
		return domain.ImportChunkResult{}, fmt.Errorf("cleanup stg_addresses: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM stg_users WHERE job_id = $1", jobID); err != nil {
		return domain.ImportChunkResult{}, fmt.Errorf("cleanup stg_users: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ImportChunkResult{}, fmt.Errorf("commit import chunk: %w", err)
	}

	return domain.ImportChunkResult{
		ImportedCount: imported,
		UpdatedCount:  updated,
		SkippedCount:  0,
	}, nil
}

func upsertUsersByExternalID(ctx context.Context, tx pgx.Tx, jobID string) (int64, int64, error) {
	rows, err := tx.Query(ctx, `
WITH staged AS (
    SELECT DISTINCT ON (external_id)
      CASE WHEN external_id ~* $2 THEN external_id::uuid ELSE NULL END AS ext_uuid,
      name,
      email,
      phone_number
    FROM stg_users
    WHERE job_id = $1 AND external_id IS NOT NULL AND external_id <> ''
    ORDER BY external_id, row_index DESC
), upserted AS (
    INSERT INTO users (id, name, email, phone_number, created_at, updated_at)
    SELECT ext_uuid, name, email, phone_number, NOW(), NOW()
    FROM staged
    WHERE ext_uuid IS NOT NULL
    ON CONFLICT (id) DO UPDATE
      SET name = EXCLUDED.name,
          email = EXCLUDED.email,
          phone_number = EXCLUDED.phone_number,
          updated_at = NOW()
    RETURNING (xmax = 0) AS inserted
)
SELECT inserted FROM upserted
`, jobID, uuidRegex)
	if err != nil {
		return 0, 0, fmt.Errorf("upsert users by external_id: %w", err)
	}
	defer rows.Close()

	return countInsertedUpdated(rows)
}

func upsertUsersByEmail(ctx context.Context, tx pgx.Tx, jobID string) (int64, int64, error) {
	rows, err := tx.Query(ctx, `
WITH staged AS (
    SELECT DISTINCT ON (email)
      name,
      email,
      phone_number
    FROM stg_users
    WHERE job_id = $1 AND (external_id IS NULL OR external_id = '' OR NOT (external_id ~* $2))
    ORDER BY email, row_index DESC
), upserted AS (
    INSERT INTO users (name, email, phone_number, created_at, updated_at)
    SELECT name, email, phone_number, NOW(), NOW()
    FROM staged
    ON CONFLICT (email) DO UPDATE
      SET name = EXCLUDED.name,
          phone_number = EXCLUDED.phone_number,
          updated_at = NOW()
    RETURNING (xmax = 0) AS inserted
)
SELECT inserted FROM upserted
`, jobID, uuidRegex)
	if err != nil {
		return 0, 0, fmt.Errorf("upsert users by email: %w", err)
	}
	defer rows.Close()

	return countInsertedUpdated(rows)
}

func replaceAddresses(ctx context.Context, tx pgx.Tx, jobID string) error {
	if _, err := tx.Exec(ctx, `
WITH affected_users AS (
    SELECT DISTINCT u.id
    FROM users u
    JOIN stg_users s
      ON s.job_id = $1
     AND (
       (CASE WHEN s.external_id ~* $2 THEN s.external_id::uuid ELSE NULL END) = u.id
       OR ((s.external_id IS NULL OR s.external_id = '' OR NOT (s.external_id ~* $2)) AND u.email = s.email)
     )
)
DELETE FROM addresses a
USING affected_users af
WHERE a.user_id = af.id
`, jobID, uuidRegex); err != nil {
		return fmt.Errorf("delete existing addresses: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO addresses (user_id, street, city, state, zip_code, country, created_at, updated_at)
SELECT
  u.id,
  a.street,
  a.city,
  a.state,
  a.zip_code,
  a.country,
  NOW(),
  NOW()
FROM stg_addresses a
JOIN users u
  ON (
    (CASE WHEN a.user_external_id ~* $2 THEN a.user_external_id::uuid ELSE NULL END) = u.id
    OR ((a.user_external_id IS NULL OR a.user_external_id = '' OR NOT (a.user_external_id ~* $2)) AND u.email = a.user_email)
  )
WHERE a.job_id = $1
`, jobID, uuidRegex); err != nil {
		return fmt.Errorf("insert replacement addresses: %w", err)
	}

	return nil
}

func countInsertedUpdated(rows pgx.Rows) (int64, int64, error) {
	var imported int64
	var updated int64

	for rows.Next() {
		var inserted bool
		if err := rows.Scan(&inserted); err != nil {
			return 0, 0, err
		}
		if inserted {
			imported++
		} else {
			updated++
		}
	}

	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	return imported, updated, nil
}

func nullableText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
