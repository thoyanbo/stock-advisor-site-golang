package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"stock-advisor-site-golang/config"
	"stock-advisor-site-golang/handlers"
	"stock-advisor-site-golang/logging"
	"stock-advisor-site-golang/middleware"
	"stock-advisor-site-golang/repository"
	"stock-advisor-site-golang/services"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present (ignored in production / Docker where env vars are set directly)
	_ = godotenv.Load()

	cfg := config.Load()
	logging.SetLevel(cfg.LogLevel)
	logging.Info("server_starting", map[string]interface{}{
		"port":         cfg.Port,
		"dataProvider": cfg.DataProvider,
		"logLevel":     cfg.LogLevel,
	})

	svc := services.NewMarketDataService(cfg)
	scenarioRepo := repository.NewScenarioRepository("")
	rl := middleware.NewRateLimiter()

	healthH := handlers.NewHealthHandler(svc)
	tickerH := handlers.NewTickerHandlers(svc, cfg)
	scenarioH := handlers.NewScenarioHandlers(scenarioRepo)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", healthH.ServeHTTP)

		api.With(rl.Enforce("tickers-search", 40, 60*time.Second)).
			Get("/tickers/search", tickerH.Search)

		api.Route("/tickers/{symbol}", func(tr chi.Router) {
			tr.With(rl.Enforce("ticker-summary", 50, 60*time.Second)).
				Get("/summary", tickerH.Summary)
			tr.With(rl.Enforce("ticker-historical", 35, 60*time.Second)).
				Get("/historical-growth", tickerH.HistoricalGrowth)
			tr.With(rl.Enforce("ticker-fmp-reference", 30, 60*time.Second)).
				Get("/fmp-reference", tickerH.FmpReference)

			tr.With(rl.Enforce("scenarios-list", 80, 60*time.Second)).
				Get("/scenarios", scenarioH.List)
			tr.With(rl.Enforce("scenarios-create", 25, 60*time.Second)).
				Post("/scenarios", scenarioH.Create)
			tr.Put("/scenarios/{scenarioId}", scenarioH.Update)
			tr.Delete("/scenarios/{scenarioId}", scenarioH.Delete)
		})

		api.With(rl.Enforce("valuation-dcf", 80, 60*time.Second)).
			Post("/valuation/dcf", handlers.DCFHandler)
		api.With(rl.Enforce("valuation-reverse-dcf", 80, 60*time.Second)).
			Post("/valuation/reverse-dcf", handlers.ReverseDCFHandler)
	})

	distPath := filepath.Join("frontend", "dist")
	if info, err := os.Stat(distPath); err == nil && info.IsDir() {
		fileServer := http.FileServer(http.Dir(distPath))
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			path := req.URL.Path
			if strings.HasPrefix(path, "/api/") {
				http.NotFound(w, req)
				return
			}
			filePath := filepath.Join(distPath, filepath.Clean(path))
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				http.ServeFile(w, req, filepath.Join(distPath, "index.html"))
				return
			}
			fileServer.ServeHTTP(w, req)
		})
	} else {
		r.Get("/*", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "stock-advisor-site-golang API is running. Build the frontend to serve the SPA.")
		})
	}

	addr := ":" + cfg.Port
	logging.Info("server_listening", map[string]interface{}{"addr": addr})
	if err := http.ListenAndServe(addr, r); err != nil {
		logging.Error("server_fatal", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}
}
