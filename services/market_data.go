package services

import (
	"stock-advisor-site-golang/config"
	"stock-advisor-site-golang/logging"
	"stock-advisor-site-golang/providers"
	"stock-advisor-site-golang/valuation"
	"sync"
	"time"
)

type DataSource string

const (
	SourceSeed         DataSource = "seed"
	SourceFMP          DataSource = "fmp"
	SourceSeedFallback DataSource = "seed-fallback"
)

type ProviderResult[T any] struct {
	Data            T
	Source          DataSource
	ProviderMessage string
}

type ttlEntry struct {
	value     interface{}
	expiresAt time.Time
}

type ttlCache struct {
	mu    sync.RWMutex
	store map[string]ttlEntry
}

func newTTLCache() *ttlCache {
	return &ttlCache{store: make(map[string]ttlEntry)}
}

func (c *ttlCache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.store[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (c *ttlCache) set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = ttlEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

type MarketDataService struct {
	cfg          *config.Config
	seedProvider *providers.SeedProvider
	cache        *ttlCache
}

func NewMarketDataService(cfg *config.Config) *MarketDataService {
	return &MarketDataService{
		cfg:          cfg,
		seedProvider: &providers.SeedProvider{},
		cache:        newTTLCache(),
	}
}

func (s *MarketDataService) createActiveProvider() providers.MarketDataProvider {
	if s.cfg.DataProvider == "fmp" && s.cfg.FMPApiKey != "" {
		return providers.NewFmpProvider(s.cfg.FMPApiKey, s.cfg.FinnhubApiKey)
	}
	return s.seedProvider
}

func (s *MarketDataService) GetProviderInfo() map[string]interface{} {
	active := s.cfg.DataProvider
	fmpConfigured := s.cfg.FMPApiKey != ""
	if active == "fmp" && !fmpConfigured {
		active = "seed"
	}
	var warnings []string
	if s.cfg.DataProvider == "fmp" && !fmpConfigured {
		warnings = append(warnings, "DATA_PROVIDER=fmp but FMP_API_KEY is empty; falling back to seed")
	}
	return map[string]interface{}{
		"configuredProvider":  s.cfg.DataProvider,
		"activeProvider":      active,
		"fmpApiKeyConfigured": fmpConfigured,
		"warnings":            warnings,
	}
}

func withFallback[T any](s *MarketDataService, fn func(providers.MarketDataProvider) (T, error)) ProviderResult[T] {
	primary := s.createActiveProvider()
	if _, isSeed := primary.(*providers.SeedProvider); isSeed {
		data, err := fn(primary)
		if err != nil {
			logging.Error("seed_provider_error", map[string]interface{}{"error": err.Error()})
		}
		return ProviderResult[T]{Data: data, Source: SourceSeed}
	}

	data, err := fn(primary)
	if err == nil {
		return ProviderResult[T]{Data: data, Source: SourceFMP}
	}

	logging.Warn("fmp_fallback_to_seed", map[string]interface{}{"error": err.Error()})
	data, seedErr := fn(s.seedProvider)
	if seedErr != nil {
		logging.Error("seed_fallback_error", map[string]interface{}{"error": seedErr.Error()})
	}
	return ProviderResult[T]{Data: data, Source: SourceSeedFallback, ProviderMessage: providers.ToProviderMessage(err)}
}

func (s *MarketDataService) SearchTickers(query string) ProviderResult[[]providers.TickerSearchResult] {
	key := "search:" + query
	if cached, ok := s.cache.get(key); ok {
		return cached.(ProviderResult[[]providers.TickerSearchResult])
	}
	result := withFallback(s, func(p providers.MarketDataProvider) ([]providers.TickerSearchResult, error) {
		return p.SearchTickers(query)
	})
	s.cache.set(key, result, 60*time.Second)
	return result
}

func (s *MarketDataService) GetTickerSummary(symbol string) ProviderResult[*providers.TickerSummary] {
	key := "summary:" + symbol
	if cached, ok := s.cache.get(key); ok {
		return cached.(ProviderResult[*providers.TickerSummary])
	}
	result := withFallback(s, func(p providers.MarketDataProvider) (*providers.TickerSummary, error) {
		return p.GetTickerSummary(symbol)
	})
	s.cache.set(key, result, 60*time.Second)
	return result
}

func (s *MarketDataService) GetTickerHistorical(symbol string) ProviderResult[*valuation.TickerHistoricalData] {
	key := "historical:" + symbol
	if cached, ok := s.cache.get(key); ok {
		return cached.(ProviderResult[*valuation.TickerHistoricalData])
	}
	result := withFallback(s, func(p providers.MarketDataProvider) (*valuation.TickerHistoricalData, error) {
		return p.GetTickerHistorical(symbol)
	})
	s.cache.set(key, result, 120*time.Second)
	return result
}
