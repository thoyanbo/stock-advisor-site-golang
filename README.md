# Stock Advisor

A small stock valuation tool: search a ticker, pull its fundamentals, and run
DCF-style valuations to estimate fair value and margin of safety. Built as a
Go API with a React/TypeScript frontend.

## Features

- **Ticker search & summary** — price, EPS, FCF/share, dividend/share, tangible
  book value/share.
- **Historical growth data** — up to 5 years of EPS, revenue, operating
  income, EBITDA, FCF, dividends, book value, and price.
- **Discounted Cash Flow (DCF)** — two-stage model (growth stage + terminal
  stage) with an optional tangible-book-value add-on. Returns fair value and
  margin of safety vs. the current price.
- **Reverse DCF** — given the current price, solves for the growth rate the
  market is implicitly pricing in.
- **FMP reference** — cross-checks your model against FMP's own published DCF
  estimate for the same ticker.
- **Scenarios** — save, update, and delete named valuation scenarios per
  ticker (stored locally as JSON).

## Data providers

Fundamentals come from one of two sources, selected by `DATA_PROVIDER`:

- `seed` (default) — a small built-in offline dataset, useful for local
  development without any API keys.
- `fmp` — live data from [Financial Modeling Prep](https://financialmodelingprep.com/),
  optionally supplemented by [Finnhub](https://finnhub.io/) to backfill
  missing historical figures. If a live request fails, the app automatically
  falls back to seed data so the UI keeps working.

## Getting started

### Backend

```bash
go run .
```

Configuration is read from environment variables (or a `.env` file — copy
`.env.example` to `.env` and fill in your own keys):

| Variable          | Default | Description                                   |
|-------------------|---------|------------------------------------------------|
| `DATA_PROVIDER`   | `seed`  | `seed` or `fmp`                                |
| `FMP_API_KEY`     | —       | required for `DATA_PROVIDER=fmp`               |
| `FINNHUB_API_KEY` | —       | optional, supplements FMP historical data       |
| `LOG_LEVEL`       | `info`  | log verbosity                                   |
| `PORT`            | `8080`  | HTTP port                                       |

The server always serves the API. If `frontend/dist` exists (i.e. the
frontend has been built), it also serves the built SPA for any non-`/api`
route.

### Frontend

```bash
cd frontend
npm install
npm run dev       # local dev server
npm run build     # production build -> frontend/dist
```

### Docker

```bash
docker build -t stock-advisor .
docker run -p 8080:8080 --env-file .env stock-advisor
```

The multi-stage `Dockerfile` builds the frontend, builds the Go binary, and
ships a slim Alpine runtime image that serves both from a single container.

## API

All endpoints are under `/api/v1`.

| Method | Path                                       | Description                          |
|--------|---------------------------------------------|---------------------------------------|
| GET    | `/health`                                   | Service health & active data provider |
| GET    | `/tickers/search?q=`                        | Search tickers                        |
| GET    | `/tickers/{symbol}/summary`                 | Latest fundamentals                   |
| GET    | `/tickers/{symbol}/historical-growth`       | Historical growth series              |
| GET    | `/tickers/{symbol}/fmp-reference`           | FMP's own DCF estimate                |
| GET    | `/tickers/{symbol}/scenarios`               | List saved scenarios                  |
| POST   | `/tickers/{symbol}/scenarios`               | Create a scenario                     |
| PUT    | `/tickers/{symbol}/scenarios/{scenarioId}`  | Update a scenario                     |
| DELETE | `/tickers/{symbol}/scenarios/{scenarioId}`  | Delete a scenario                     |
| POST   | `/valuation/dcf`                            | Compute a DCF valuation               |
| POST   | `/valuation/reverse-dcf`                    | Compute a reverse DCF valuation       |

All routes are rate-limited per-endpoint.

## Project layout

```
config/       env-based configuration
data/seed/    offline seed dataset
handlers/     HTTP handlers
logging/      structured logger
middleware/   request ID + rate limiting
providers/    market data providers (seed, FMP + Finnhub)
repository/   local JSON-backed scenario storage
services/     market data service (caching + provider fallback)
valuation/    DCF / reverse DCF / growth calculations
frontend/     React + TypeScript + Vite SPA
```
