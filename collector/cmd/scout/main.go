package main

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"jobpulse/collector/internal/events"
	"jobpulse/collector/internal/sources"
	"jobpulse/collector/internal/storage"
)

type scout struct {
	ctx       context.Context
	store     *storage.Store
	publisher *events.Publisher
	client    *http.Client
	boards    []string
}

//go:embed dashboard.html
var dashboardHTML embed.FS

type dashboardData struct {
	Jobs   []storage.ScoredPosting
	Filter string
	Error  string
}

func (s *scout) poll(ctx context.Context, board string) {
	jobs, err := sources.FetchGreenhouse(ctx, s.client, board)
	if err != nil {
		slog.Error("poll failed", "board", board, "error", err)
		return
	}
	for i := range jobs {
		jobs[i].DiscoveredAt = time.Now().UTC()
		isNew, err := s.store.Save(jobs[i])
		if err != nil {
			slog.Error("save failed", "posting", jobs[i].ID, "error", err)
			continue
		}
		if isNew {
			if err := s.publisher.Publish(ctx, jobs[i]); err != nil {
				slog.Error("publish failed", "posting", jobs[i].ID, "error", err)
				continue
			}
			slog.Info("published posting", "posting", jobs[i].ID)
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	boards := splitEnv("GREENHOUSE_BOARDS")
	if len(boards) == 0 {
		slog.Warn("GREENHOUSE_BOARDS is empty; use /trigger/{source} after configuring it")
	}
	dbPath := envOr("DATABASE_PATH", "./data/jobpulse.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	rdb := redis.NewClient(&redis.Options{Addr: envOr("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("connect to Redis", "error", err)
		os.Exit(1)
	}
	s := &scout{ctx: ctx, store: store, publisher: events.NewPublisher(rdb, envOr("REDIS_STREAM", "job.posting.discovered")), client: &http.Client{Timeout: 30 * time.Second}, boards: boards}
	interval, err := time.ParseDuration(envOr("POLL_INTERVAL", "15m"))
	if err != nil {
		slog.Error("invalid POLL_INTERVAL", "error", err)
		os.Exit(1)
	}
	var wg sync.WaitGroup
	for _, board := range boards {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.poll(ctx, board)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.poll(ctx, board)
				}
			}
		}()
	}
	h := http.NewServeMux()
	dashboard, err := template.ParseFS(dashboardHTML, "dashboard.html")
	if err != nil {
		slog.Error("load dashboard", "error", err)
		os.Exit(1)
	}
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("recommendation")
		if filter == "" {
			filter = "review"
		}
		if filter == "all" {
			filter = ""
		}
		jobs, err := s.store.ListScored(100, filter)
		if err != nil {
			http.Error(w, "could not load dashboard", http.StatusInternalServerError)
			return
		}
		if err := dashboard.Execute(w, dashboardData{Jobs: jobs, Filter: filter}); err != nil {
			slog.Error("render dashboard", "error", err)
		}
	})
	h.HandleFunc("/api/postings", func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if value := r.URL.Query().Get("limit"); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				limit = parsed
			}
		}
		jobs, err := s.store.ListScored(limit, r.URL.Query().Get("recommendation"))
		if err != nil {
			http.Error(w, "could not load postings", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jobs)
	})
	h.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	h.HandleFunc("/trigger/greenhouse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		for _, board := range s.boards {
			go s.poll(s.ctx, board)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	server := &http.Server{Addr: envOr("HTTP_ADDR", ":8080"), Handler: h}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server", "error", err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	wg.Wait()
}

func splitEnv(name string) []string {
	var result []string
	for _, value := range strings.Split(os.Getenv(name), ",") {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
