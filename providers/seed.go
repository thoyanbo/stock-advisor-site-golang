package providers

import (
	"stock-advisor-site-golang/data/seed"
	"stock-advisor-site-golang/valuation"
)

type SeedProvider struct{}

func (p *SeedProvider) SearchTickers(query string) ([]TickerSearchResult, error) {
	seeds := seed.SearchSeeds(query)
	results := make([]TickerSearchResult, 0, len(seeds))
	for _, s := range seeds {
		results = append(results, TickerSearchResult{
			Symbol:   s.Symbol,
			Name:     s.Name,
			Exchange: s.Exchange,
		})
	}
	return results, nil
}

func (p *SeedProvider) GetTickerSummary(symbol string) (*TickerSummary, error) {
	t := seed.FindBySymbol(symbol)
	if t == nil {
		return nil, nil
	}
	return &TickerSummary{
		Symbol:   t.Symbol,
		Name:     t.Name,
		Exchange: t.Exchange,
		Currency: t.Currency,
		Price:    t.Price,
		ValuationBases: ValuationBases{
			EPS:      t.EPS,
			FCF:      t.FCFPerShare,
			Dividend: t.DividendPerShare,
		},
		TangibleBookValuePerShare: t.TangibleBookValuePerShare,
	}, nil
}

func (p *SeedProvider) GetTickerHistorical(symbol string) (*valuation.TickerHistoricalData, error) {
	t := seed.FindBySymbol(symbol)
	if t == nil {
		return nil, nil
	}
	hist := t.Historical
	hist.Diagnostics = &valuation.HistoricalDiagnostics{SupplementStatus: "none"}
	return &hist, nil
}
