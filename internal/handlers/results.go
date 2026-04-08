package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"chperf.example.com/internal/stats"
	"chperf.example.com/web/templates"
)

// Results renders the results page with charts.
func Results(repo *stats.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid run id")
			return
		}
		run, err := repo.GetRun(id)
		if err != nil || run == nil {
			c.String(http.StatusNotFound, "run not found")
			return
		}
		queries, err := repo.GetQueries(id)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to get queries: %v", err)
			return
		}

		chartData := buildChartData(repo, queries, id)
		chartDataJSON, _ := json.Marshal(chartData)

		c.HTML(http.StatusOK, "", templates.Results(run, queries, chartData, string(chartDataJSON)))
	}
}

// ExportCSV streams the run results as CSV, including a per-query summary.
func ExportCSV(repo *stats.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid run id")
			return
		}
		run, err := repo.GetRun(id)
		if err != nil || run == nil {
			c.String(http.StatusNotFound, "run not found")
			return
		}
		rows, err := repo.GetResultsForExport(id)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to get results: %v", err)
			return
		}
		queries, err := repo.GetQueries(id)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to get queries: %v", err)
			return
		}

		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=\"run_"+strconv.FormatInt(id, 10)+"_results.csv\"")

		ff := func(f float64) string { return fmt.Sprintf("%.2f", f) }

		w := csv.NewWriter(c.Writer)

		// Per-query summary section
		_ = w.Write([]string{"## Query Latency Summary"})
		_ = w.Write([]string{"query_id", "sql_preview", "ok_count", "error_count", "avg_ms", "min_ms", "max_ms", "p50_ms", "p75_ms", "p90_ms", "p95_ms", "p99_ms", "error_messages"})
		for _, q := range queries {
			vals, _ := repo.GetRunTimeMsForQuery(q.ID)
			errSummary, _ := repo.GetErrorSummaryForQuery(q.ID)
			var errCount int
			var errMsgs string
			if errSummary != nil {
				errCount = errSummary.Count
				errMsgs = strings.Join(errSummary.Messages, " | ")
			}
			preview := q.SQLText
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", " ")
			_ = w.Write([]string{
				strconv.FormatInt(q.ID, 10),
				preview,
				strconv.Itoa(len(vals)),
				strconv.Itoa(errCount),
				ff(stats.Average(vals)),
				ff(stats.MinVal(vals)),
				ff(stats.MaxVal(vals)),
				ff(stats.Percentile(vals, 50)),
				ff(stats.Percentile(vals, 75)),
				ff(stats.Percentile(vals, 90)),
				ff(stats.Percentile(vals, 95)),
				ff(stats.Percentile(vals, 99)),
				errMsgs,
			})
		}

		// Blank separator
		_ = w.Write([]string{})

		// Raw execution data
		_ = w.Write([]string{"## Raw Execution Data"})
		_ = w.Write([]string{"run_id", "run_name", "query_id", "sql_text", "run_time_ms", "rows_affected", "error_message", "executed_at", "worker_id"})
		for _, row := range rows {
			sqlPreview := row.SQLText
			if len(sqlPreview) > 200 {
				sqlPreview = sqlPreview[:200] + "..."
			}
			_ = w.Write([]string{
				strconv.FormatInt(row.RunID, 10),
				row.RunName,
				strconv.FormatInt(row.QueryID, 10),
				sqlPreview,
				strconv.FormatFloat(row.RunTimeMs, 'f', -1, 64),
				strconv.FormatInt(row.RowsAffected, 10),
				row.ErrorMessage,
				row.ExecutedAt.Format("2006-01-02 15:04:05"),
				strconv.Itoa(row.WorkerID),
			})
		}
		w.Flush()
	}
}

func buildChartData(repo *stats.Repository, queries []stats.Query, runID int64) templates.ChartData {
	var perQuery []templates.QueryPercentiles
	var labels []string
	var timeseries []templates.QueryTimeseries

	for _, q := range queries {
		vals, _ := repo.GetRunTimeMsForQuery(q.ID)
		p := stats.Percentiles{
			P50: stats.Percentile(vals, 50),
			P75: stats.Percentile(vals, 75),
			P90: stats.Percentile(vals, 90),
			P95: stats.Percentile(vals, 95),
			P99: stats.Percentile(vals, 99),
		}
		preview := q.SQLText
		if len(preview) > 50 {
			preview = strings.TrimSpace(preview[:50]) + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " ")
		labels = append(labels, preview)
		errSummary, _ := repo.GetErrorSummaryForQuery(q.ID)
		var errCount int
		var errMsgs []string
		if errSummary != nil {
			errCount = errSummary.Count
			errMsgs = errSummary.Messages
		}

		perQuery = append(perQuery, templates.QueryPercentiles{
			QueryID:       q.ID,
			SQLPreview:    preview,
			Percentiles:   p,
			Avg:           stats.Average(vals),
			Min:           stats.MinVal(vals),
			Max:           stats.MaxVal(vals),
			Count:         len(vals),
			ErrorCount:    errCount,
			ErrorMessages: errMsgs,
		})

		pts, _ := repo.GetTimeseriesForQuery(q.ID)
		timeseries = append(timeseries, templates.QueryTimeseries{
			QueryID:    q.ID,
			SQLPreview: preview,
			Points:     pts,
		})
	}

	overallVals, _ := repo.GetRunTimeMsForRun(runID)
	overall := stats.Percentiles{
		P50: stats.Percentile(overallVals, 50),
		P75: stats.Percentile(overallVals, 75),
		P90: stats.Percentile(overallVals, 90),
		P95: stats.Percentile(overallVals, 95),
		P99: stats.Percentile(overallVals, 99),
	}

	overallErrCount, _ := repo.GetErrorCountForRun(runID)

	return templates.ChartData{
		PerQuery:          perQuery,
		Overall:           overall,
		OverallAvg:        stats.Average(overallVals),
		OverallMin:        stats.MinVal(overallVals),
		OverallMax:        stats.MaxVal(overallVals),
		OverallCount:      len(overallVals),
		OverallErrorCount: overallErrCount,
		QueryLabels:       labels,
		Timeseries:        timeseries,
	}
}
