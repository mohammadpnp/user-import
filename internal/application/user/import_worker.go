package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
)

const maxStoredFailures = 100

type ImportSource interface {
	Open(ctx context.Context, sourcePath string) (io.ReadCloser, error)
}

type ImportChunkResult = domain.ImportChunkResult

type importChunker interface {
	ImportChunk(ctx context.Context, jobID string, users []domain.User) (ImportChunkResult, error)
}

type importWorkerJobRepo interface {
	ClaimNext(ctx context.Context, leaseDuration time.Duration) (*domain.ImportJob, error)
	Heartbeat(ctx context.Context, jobID string, leaseDuration time.Duration) error
	UpdateProgress(ctx context.Context, jobID string, progress domain.ImportProgress) error
	Complete(ctx context.Context, jobID string, summary domain.ImportSummary) error
	Requeue(ctx context.Context, jobID string, reason string) error
	Fail(ctx context.Context, jobID string, reason string) error
}

type ImportWorkerConfig struct {
	Workers           int
	ChunkSize         int
	PollInterval      time.Duration
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
}

type ImportWorker struct {
	repo     importWorkerJobRepo
	source   ImportSource
	importer importChunker
	cfg      ImportWorkerConfig

	once sync.Once
}

func NewImportWorker(repo importWorkerJobRepo, source ImportSource, importer importChunker, cfg ImportWorkerConfig) *ImportWorker {
	if cfg.Workers <= 0 {
		cfg.Workers = 10
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 10000
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = 60 * time.Second
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = cfg.LeaseDuration / 2
	}

	return &ImportWorker{
		repo:     repo,
		source:   source,
		importer: importer,
		cfg:      cfg,
	}
}

func (w *ImportWorker) Start(ctx context.Context) {
	w.once.Do(func() {
		for i := 0; i < w.cfg.Workers; i++ {
			go w.workerLoop(ctx)
		}
	})
}

func (w *ImportWorker) workerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := w.repo.ClaimNext(ctx, w.cfg.LeaseDuration)
		if err != nil {
			log.Printf("claim next import job failed: %v", err)
			if !sleepWithContext(ctx, w.cfg.PollInterval) {
				return
			}
			continue
		}

		if job == nil {
			if !sleepWithContext(ctx, w.cfg.PollInterval) {
				return
			}
			continue
		}

		if err := w.ProcessJob(ctx, *job); err != nil {
			log.Printf("process import job %s failed: %v", job.ID, err)
		}
	}
}

func (w *ImportWorker) ProcessJob(ctx context.Context, job domain.ImportJob) error {
	reader, err := w.source.Open(ctx, job.SourcePath)
	if err != nil {
		return w.onProcessingError(ctx, job, fmt.Errorf("open import source: %w", err))
	}
	defer reader.Close()

	dec := json.NewDecoder(reader)

	token, err := dec.Token()
	if err != nil {
		return w.onProcessingError(ctx, job, fmt.Errorf("read json start token: %w", err))
	}

	delim, ok := token.(json.Delim)
	if !ok || delim != '[' {
		return w.onProcessingError(ctx, job, errors.New("import payload must be a JSON array"))
	}

	ticker := time.NewTicker(w.cfg.HeartbeatInterval)
	defer ticker.Stop()

	summary := domain.ImportSummary{}
	chunk := make([]domain.User, 0, w.cfg.ChunkSize)

	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}

		result, importErr := w.importer.ImportChunk(ctx, job.ID, chunk)
		if importErr != nil {
			return importErr
		}

		summary.ImportedCount += result.ImportedCount
		summary.UpdatedCount += result.UpdatedCount
		summary.SkippedCount += result.SkippedCount

		if err := w.repo.UpdateProgress(ctx, job.ID, domain.ImportProgress{
			ProcessedCount: summary.ProcessedCount,
			ImportedCount:  summary.ImportedCount,
			UpdatedCount:   summary.UpdatedCount,
			SkippedCount:   summary.SkippedCount,
			FailedCount:    summary.FailedCount,
		}); err != nil {
			return err
		}

		chunk = chunk[:0]
		return nil
	}

	var rowIndex int64
	for dec.More() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.repo.Heartbeat(ctx, job.ID, w.cfg.LeaseDuration); err != nil {
				return w.onProcessingError(ctx, job, fmt.Errorf("heartbeat: %w", err))
			}
		default:
		}

		var raw rawUser
		if err := dec.Decode(&raw); err != nil {
			return w.onProcessingError(ctx, job, fmt.Errorf("decode user at index %d: %w", rowIndex, err))
		}

		summary.ProcessedCount++

		userAggregate, validationErr := raw.toDomain()
		if validationErr != nil {
			summary.FailedCount++
			summary.SkippedCount++
			if len(summary.Failures) < maxStoredFailures {
				summary.Failures = append(summary.Failures, domain.ImportFailure{
					RowIndex: rowIndex,
					Reason:   validationErr.Error(),
				})
			}
			rowIndex++
			continue
		}

		chunk = append(chunk, userAggregate)
		if len(chunk) >= w.cfg.ChunkSize {
			if err := flush(); err != nil {
				return w.onProcessingError(ctx, job, fmt.Errorf("flush chunk: %w", err))
			}
			if err := w.repo.Heartbeat(ctx, job.ID, w.cfg.LeaseDuration); err != nil {
				return w.onProcessingError(ctx, job, fmt.Errorf("heartbeat after flush: %w", err))
			}
		}

		rowIndex++
	}

	if _, err := dec.Token(); err != nil {
		return w.onProcessingError(ctx, job, fmt.Errorf("read json end token: %w", err))
	}

	if err := flush(); err != nil {
		return w.onProcessingError(ctx, job, fmt.Errorf("flush last chunk: %w", err))
	}

	if err := w.repo.UpdateProgress(ctx, job.ID, domain.ImportProgress{
		ProcessedCount: summary.ProcessedCount,
		ImportedCount:  summary.ImportedCount,
		UpdatedCount:   summary.UpdatedCount,
		SkippedCount:   summary.SkippedCount,
		FailedCount:    summary.FailedCount,
	}); err != nil {
		return w.onProcessingError(ctx, job, fmt.Errorf("update final progress: %w", err))
	}

	if err := w.repo.Complete(ctx, job.ID, summary); err != nil {
		return w.onProcessingError(ctx, job, fmt.Errorf("complete job: %w", err))
	}

	return nil
}

func (w *ImportWorker) onProcessingError(ctx context.Context, job domain.ImportJob, err error) error {
	reason := truncateReason(err.Error())
	if job.Attempts < job.MaxAttempts {
		if requeueErr := w.repo.Requeue(ctx, job.ID, reason); requeueErr != nil {
			return fmt.Errorf("%v; requeue failed: %w", err, requeueErr)
		}
		return err
	}

	if failErr := w.repo.Fail(ctx, job.ID, reason); failErr != nil {
		return fmt.Errorf("%v; fail update failed: %w", err, failErr)
	}
	return err
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func truncateReason(reason string) string {
	const maxLen = 1000
	reason = strings.TrimSpace(reason)
	if len(reason) <= maxLen {
		return reason
	}
	return reason[:maxLen]
}

type rawAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

type rawUser struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Email       string       `json:"email"`
	PhoneNumber string       `json:"phone_number"`
	Addresses   []rawAddress `json:"addresses"`
}

func (u rawUser) toDomain() (domain.User, error) {
	addresses := make([]domain.Address, 0, len(u.Addresses))
	for _, address := range u.Addresses {
		addresses = append(addresses, domain.Address{
			Street:  address.Street,
			City:    address.City,
			State:   address.State,
			ZipCode: address.ZipCode,
			Country: address.Country,
		})
	}

	return domain.NewUser(u.ID, u.Name, u.Email, u.PhoneNumber, addresses)
}
