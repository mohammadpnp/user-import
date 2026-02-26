package user

import "context"

type ImportJobRepository interface {
	Enqueue(ctx context.Context, sourcePath string) (string, error)
}
