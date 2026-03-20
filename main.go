package main

import (
	"log"
	"net/http"

	"chperf.example.com/config"
	"chperf.example.com/internal/connections"
	"chperf.example.com/internal/db"
	"chperf.example.com/internal/handlers"
	"chperf.example.com/internal/loadrunner"
	"chperf.example.com/internal/renderer"
	"chperf.example.com/internal/stats"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	sqliteDB, err := db.InitSQLite(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("init sqlite: %v", err)
	}
	defer sqliteDB.Close()

	repo := stats.NewRepository(sqliteDB)
	connRepo := connections.NewRepository(sqliteDB)
	runner := loadrunner.NewRunner(repo)

	engine := gin.Default()
	engine.HTMLRender = &renderer.HTMLTemplRenderer{}

	engine.Static("/static", "./web/static")

	engine.GET("/", handlers.Home(repo))
	engine.GET("/connections", handlers.ConnectionsList(connRepo))
	engine.GET("/connections/new", handlers.ConnectionForm())
	engine.POST("/connections", handlers.CreateConnection(connRepo))
	engine.GET("/connections/:id/edit", handlers.ConnectionEditForm(connRepo))
	engine.POST("/connections/:id", handlers.UpdateConnection(connRepo))
	engine.POST("/connections/:id/delete", handlers.DeleteConnection(connRepo))
	engine.POST("/connections/:id/test", handlers.TestConnection(connRepo))
	engine.POST("/connections/test", handlers.TestConnectionDSN())

	engine.GET("/runs/new", handlers.NewRunForm(connRepo, cfg))
	engine.POST("/runs", handlers.CreateRun(repo, connRepo, runner, cfg))
	engine.GET("/runs/:id", handlers.RunStatus(repo))
	engine.POST("/runs/:id/delete", handlers.DeleteRun(repo))
	engine.GET("/runs/:id/results", handlers.Results(repo))
	engine.GET("/runs/:id/export", handlers.ExportCSV(repo))
	engine.GET("/api/runs/:id/status", handlers.RunStatusAPI(repo))

	log.Println("chPerf listening on :8085")
	if err := engine.Run(":8085"); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
