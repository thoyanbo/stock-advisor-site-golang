package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"stock-advisor-site-golang/config"
	"stock-advisor-site-golang/services"
	"stock-advisor-site-golang/valuation"
	"strings"

	"github.com/go-chi/chi/v5"
)

type TickerHandlers struct {
	svc *services.MarketDataService
	cfg *config.Config
}

func NewTickerHandlers(svc *services.MarketDataService, cfg *config.Config) *TickerHandlers {
	return &TickerHandlers{svc: svc, cfg: cfg}
}

func (h *TickerHandlers) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": []interface{}{}})
		return
	}
	result := h.svc.SearchTickers(query)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results":         result.Data,
		"provider":        result.Source,
		"providerMessage": result.ProviderMessage,
	})
}

func (h *TickerHandlers) Summary(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	result := h.svc.GetTickerSummary(symbol)
	if result.Data == nil {
		writeError(w, http.StatusNotFound, "Ticker not found.")
		return
	}
	resp := map[string]interface{}{
		"symbol":                    result.Data.Symbol,
		"name":                      result.Data.Name,
		"exchange":                  result.Data.Exchange,
		"currency":                  result.Data.Currency,
		"price":                     result.Data.Price,
		"valuationBases":            result.Data.ValuationBases,
		"tangibleBookValuePerShare": result.Data.TangibleBookValuePerShare,
		"provider":                  result.Source,
		"providerMessage":           result.ProviderMessage,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TickerHandlers) HistoricalGrowth(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	summaryResult := h.svc.GetTickerSummary(symbol)
	historicalResult := h.svc.GetTickerHistorical(symbol)

	if summaryResult.Data == nil || historicalResult.Data == nil {
		writeError(w, http.StatusNotFound, "Ticker not found.")
		return
	}

	rows := valuation.BuildHistoricalGrowthRows(historicalResult.Data)
	dataQuality := valuation.AssessHistoricalDataQuality(rows)
	provider := summaryResult.Source
	if historicalResult.Source == services.SourceSeedFallback {
		provider = historicalResult.Source
	}
	providerMessage := historicalResult.ProviderMessage
	if providerMessage == "" {
		providerMessage = summaryResult.ProviderMessage
	}

	var diag interface{}
	if historicalResult.Data.Diagnostics != nil {
		diag = historicalResult.Data.Diagnostics
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"symbol":                summaryResult.Data.Symbol,
		"rows":                  rows,
		"historicalSeries":      historicalResult.Data,
		"dataQuality":           dataQuality,
		"provider":              provider,
		"providerMessage":       providerMessage,
		"historicalDiagnostics": diag,
	})
}

type fmpDCFRow struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`
	DCF    float64 `json:"dcf"`
}

type fmpLeveredDCFRow struct {
	Symbol     string  `json:"symbol"`
	Date       string  `json:"date"`
	DCF        float64 `json:"dcf"`
	LeveredDCF float64 `json:"leveredDcf"`
}

func (h *TickerHandlers) FmpReference(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	apiKey := h.cfg.FMPApiKey

	if apiKey == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"symbol":    symbol,
			"provider":  "fmp-reference",
			"available": false,
			"message":   "FMP reference unavailable: FMP_API_KEY is not configured.",
		})
		return
	}

	fmpBase := "https://financialmodelingprep.com/stable"
	enc := url.QueryEscape(symbol)

	fetchFMP := func(path string) ([]byte, error) {
		sep := "&"
		if !strings.Contains(path, "?") {
			sep = "?"
		}
		u := fmt.Sprintf("%s%s%sapikey=%s", fmpBase, path, sep, url.QueryEscape(apiKey))
		resp, err := http.Get(u)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return nil, fmt.Errorf("FMP request failed (%d): %s", resp.StatusCode, string(body))
		}
		return io.ReadAll(resp.Body)
	}

	type fetchResult struct {
		data []byte
		err  error
	}
	ch := make(chan fetchResult, 2)
	go func() {
		d, e := fetchFMP(fmt.Sprintf("/discounted-cash-flow?symbol=%s", enc))
		ch <- fetchResult{d, e}
	}()
	go func() {
		d, e := fetchFMP(fmt.Sprintf("/levered-discounted-cash-flow?symbol=%s", enc))
		ch <- fetchResult{d, e}
	}()

	r1 := <-ch
	r2 := <-ch

	if r1.err != nil && r2.err != nil {
		msg := r1.err.Error()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"symbol":    symbol,
			"provider":  "fmp-reference",
			"available": false,
			"message":   msg,
		})
		return
	}

	var dcfRows []fmpDCFRow
	var leveredRows []fmpLeveredDCFRow
	if r1.data != nil {
		json.Unmarshal(r1.data, &dcfRows)
	}
	if r2.data != nil {
		json.Unmarshal(r2.data, &leveredRows)
	}
	if len(dcfRows) == 0 && len(leveredRows) == 0 {
		json.Unmarshal(r2.data, &dcfRows)
		json.Unmarshal(r1.data, &leveredRows)
	}

	resp := map[string]interface{}{
		"symbol":    symbol,
		"provider":  "fmp-reference",
		"available": true,
	}

	if len(dcfRows) > 0 {
		if v := dcfRows[0].DCF; math.IsInf(v, 0) || math.IsNaN(v) {
			// skip
		} else {
			resp["dcfValue"] = v
		}
		if dcfRows[0].Date != "" {
			resp["asOfDate"] = dcfRows[0].Date
		}
	}
	if len(leveredRows) > 0 {
		lv := leveredRows[0].LeveredDCF
		if lv == 0 {
			lv = leveredRows[0].DCF
		}
		if !math.IsInf(lv, 0) && !math.IsNaN(lv) {
			resp["leveredDcfValue"] = lv
		}
		if _, ok := resp["asOfDate"]; !ok && leveredRows[0].Date != "" {
			resp["asOfDate"] = leveredRows[0].Date
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
