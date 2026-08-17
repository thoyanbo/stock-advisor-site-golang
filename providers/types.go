package providers

import "stock-advisor-site-golang/valuation"

type TickerSearchResult struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
}

type ValuationBases struct {
	EPS      float64 `json:"eps"`
	FCF      float64 `json:"fcf"`
	Dividend float64 `json:"dividend"`
}

type TickerSummary struct {
	Symbol                    string         `json:"symbol"`
	Name                      string         `json:"name"`
	Exchange                  string         `json:"exchange"`
	Currency                  string         `json:"currency"`
	Price                     float64        `json:"price"`
	ValuationBases            ValuationBases `json:"valuationBases"`
	TangibleBookValuePerShare float64        `json:"tangibleBookValuePerShare"`
}

type MarketDataProvider interface {
	SearchTickers(query string) ([]TickerSearchResult, error)
	GetTickerSummary(symbol string) (*TickerSummary, error)
	GetTickerHistorical(symbol string) (*valuation.TickerHistoricalData, error)
}
