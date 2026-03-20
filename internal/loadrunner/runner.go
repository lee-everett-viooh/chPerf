package loadrunner

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"chperf.example.com/internal/clickhouse"
	"chperf.example.com/internal/stats"
)

// RunConfig holds parameters for a load test run.
type RunConfig struct {
	RunID                int64
	Queries              []stats.Query
	Mode                 string // "sequential" or "random"
	Concurrency          int
	Iterations           int
	DurationSec          int
	WaitBetweenQueriesMs int
	DSN                  string
	TLSEnabled           bool
}

// Runner executes load tests against ClickHouse.
type Runner struct {
	repo *stats.Repository
}

// NewRunner creates a new load runner.
func NewRunner(repo *stats.Repository) *Runner {
	return &Runner{repo: repo}
}

// Run starts the load test asynchronously. It spawns workers and returns immediately.
func (r *Runner) Run(ctx context.Context, cfg RunConfig) {
	go r.run(ctx, cfg)
}

func (r *Runner) run(ctx context.Context, cfg RunConfig) {
	defer func() {
		_ = r.repo.UpdateRunStatus(cfg.RunID, "completed")
	}()

	if len(cfg.Queries) == 0 {
		_ = r.repo.UpdateRunStatus(cfg.RunID, "failed")
		return
	}

	// Build query ID list for random access
	queryIDs := make([]int64, len(cfg.Queries))
	for i, q := range cfg.Queries {
		queryIDs[i] = q.ID
	}

	// Determine stop condition
	useDuration := cfg.DurationSec > 0
	endTime := time.Now().Add(time.Duration(cfg.DurationSec) * time.Second)

	var wg sync.WaitGroup
	for w := 0; w < cfg.Concurrency; w++ {
		workerID := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.runWorker(ctx, cfg, workerID, queryIDs, useDuration, endTime)
		}()
	}
	wg.Wait()
}

func (r *Runner) runWorker(ctx context.Context, cfg RunConfig, workerID int, queryIDs []int64, useDuration bool, endTime time.Time) {
	client, err := clickhouse.NewClient(ctx, cfg.DSN, cfg.TLSEnabled)
	if err != nil {
		_ = r.repo.UpdateRunStatus(cfg.RunID, "failed")
		return
	}
	defer client.Close()

	iterations := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if useDuration && time.Now().After(endTime) {
			return
		}
		if !useDuration && iterations >= cfg.Iterations {
			return
		}
		iterations++

		var qid int64
		if cfg.Mode == "random" {
			qid = queryIDs[rand.Intn(len(queryIDs))]
		} else {
			qid = queryIDs[iterations%len(queryIDs)]
		}

		// Find query text
		var sqlText string
		for _, q := range cfg.Queries {
			if q.ID == qid {
				sqlText = q.SQLText
				break
			}
		}
		if sqlText == "" {
			continue
		}

		result := client.Execute(ctx, sqlText)

		var errMsg *string
		if result.Error != nil {
			s := result.Error.Error()
			if len(s) > 500 {
				s = s[:500] + "..."
			}
			errMsg = &s
		}

		_ = r.repo.InsertQueryResult(qid, result.RunTimeMs, result.RowsAffected, errMsg, workerID)

		if cfg.WaitBetweenQueriesMs > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(cfg.WaitBetweenQueriesMs) * time.Millisecond):
			}
		}
	}
}

// RunSequentialMode runs queries in strict order: each worker executes the full sequence.
// Used when mode is "sequential" - each worker runs query 0, then 1, then 2, etc.
func (r *Runner) RunSequentialMode(ctx context.Context, cfg RunConfig) {
	// For sequential, each worker runs the full list in order, iterations times
	go func() {
		defer func() {
			_ = r.repo.UpdateRunStatus(cfg.RunID, "completed")
		}()

		if len(cfg.Queries) == 0 {
			_ = r.repo.UpdateRunStatus(cfg.RunID, "failed")
			return
		}

		useDuration := cfg.DurationSec > 0
		endTime := time.Now().Add(time.Duration(cfg.DurationSec) * time.Second)

		var wg sync.WaitGroup
		for w := 0; w < cfg.Concurrency; w++ {
			workerID := w
			wg.Add(1)
			go func() {
				defer wg.Done()
				r.runSequentialWorker(ctx, cfg, workerID, useDuration, endTime)
			}()
		}
		wg.Wait()
	}()
}

func (r *Runner) runSequentialWorker(ctx context.Context, cfg RunConfig, workerID int, useDuration bool, endTime time.Time) {
	client, err := clickhouse.NewClient(ctx, cfg.DSN, cfg.TLSEnabled)
	if err != nil {
		return
	}
	defer client.Close()

	iterations := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if useDuration && time.Now().After(endTime) {
			return
		}
		if !useDuration && iterations >= cfg.Iterations {
			return
		}
		iterations++

		for _, q := range cfg.Queries {
			select {
			case <-ctx.Done():
				return
			default:
			}
			result := client.Execute(ctx, q.SQLText)
			var errMsg *string
			if result.Error != nil {
				s := result.Error.Error()
				if len(s) > 500 {
					s = s[:500] + "..."
				}
				errMsg = &s
			}
			_ = r.repo.InsertQueryResult(q.ID, result.RunTimeMs, result.RowsAffected, errMsg, workerID)

			if cfg.WaitBetweenQueriesMs > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.WaitBetweenQueriesMs) * time.Millisecond):
				}
			}
		}
	}
}

// StartRun is the main entry point - starts a run with the given config.
func (r *Runner) StartRun(ctx context.Context, cfg RunConfig) error {
	if len(cfg.Queries) == 0 {
		return fmt.Errorf("no queries to run")
	}

	if cfg.Mode == "sequential" {
		r.RunSequentialMode(ctx, cfg)
	} else {
		r.Run(ctx, cfg)
	}
	return nil
}
