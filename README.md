# chPerf - ClickHouse Load Testing Application

A Go application for load testing ClickHouse databases. Submit SQL statements, run them sequentially or randomly with configurable concurrency, record statistics in SQLite, and visualize results with percentile charts.

## Features

- **SQL submission**: Paste multiple SQL statements (one per line)
- **Execution modes**: Sequential (run in order) or Random (pick random query each time)
- **Concurrency**: Simulate multiple concurrent users
- **Statistics**: Record run time, rows affected, errors per execution
- **Charts**: P50, P75, P90, P99 percentiles per query and overall
- **CSV export**: Download all results for analysis in Excel

## Tech Stack

- Go with Gin, templ, ClickHouse driver, SQLite
- DaisyUI + Tailwind CSS (via CDN)
- Chart.js for percentile visualization

## Quick Start

### With Docker (no Go installation required)

```bash
docker compose up --build
```

Or without Compose:

```bash
docker build -t chperf .
docker run -p 8085:8085 -v chperf-data:/app/data chperf
```

### Without Docker

```bash
# Install templ (optional, for template changes)
go install github.com/a-h/templ/cmd/templ@latest

# Generate templates and run
templ generate
go run .
```

Open http://localhost:8085

## Configuration

Environment variables:

- `CLICKHOUSE_DSN` - Default ClickHouse connection (default: `clickhouse://127.0.0.1:9000/default`)
- `SQLITE_PATH` - SQLite database path (default: `./chperf.db`)
- `DEFAULT_CONCURRENCY` - Default worker count (default: 5)
- `MAX_CONCURRENCY` - Max workers (default: 100)
- `DEFAULT_ITERATIONS` - Iterations per worker when duration=0 (default: 100)
- `DEFAULT_DURATION_SEC` - Default run duration in seconds (default: 60)

## Usage

1. **New Run**: Click "New Load Test", enter run name, ClickHouse DSN, and SQL statements
2. **Concurrency**: Number of goroutines simulating concurrent users
3. **Iterations**: Each worker runs this many iterations (or use duration)
4. **Duration**: Run for N seconds (0 = use iterations)
5. **View Results**: After completion, view charts and export CSV

## Project Structure

```
chPerf/
├── main.go
├── config/           # Configuration
├── internal/
│   ├── db/           # SQLite schema and init
│   ├── clickhouse/   # ClickHouse client
│   ├── loadrunner/   # Load test execution
│   ├── stats/        # SQLite repository
│   ├── handlers/     # HTTP handlers
│   └── renderer/     # Gin templ renderer
└── web/templates/    # Templ components
```
