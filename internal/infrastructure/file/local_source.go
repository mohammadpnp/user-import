package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalSource struct {
	BaseDir string
}

func NewLocalSource(baseDir string) *LocalSource {
	if baseDir == "" {
		baseDir = "."
	}
	return &LocalSource{BaseDir: baseDir}
}

func (s *LocalSource) Open(ctx context.Context, sourcePath string) (io.ReadCloser, error) {
	_ = ctx

	path := sourcePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.BaseDir, sourcePath)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", path, err)
	}
	return file, nil
}
