package user

type ImportJob struct {
	ID          string
	SourcePath  string
	Status      string
	Attempts    int
	MaxAttempts int
}

type ImportFailure struct {
	RowIndex int64
	Reason   string
}

type ImportProgress struct {
	ProcessedCount int64
	ImportedCount  int64
	UpdatedCount   int64
	SkippedCount   int64
	FailedCount    int64
}

type ImportSummary struct {
	ProcessedCount int64
	ImportedCount  int64
	UpdatedCount   int64
	SkippedCount   int64
	FailedCount    int64
	Failures       []ImportFailure
}
