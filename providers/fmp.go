package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"stock-advisor-site-golang/valuation"
	"strings"
	"time"
)

const (
	fmpBaseURL       = "https://financialmodelingprep.com/stable"
	finnhubBaseURL   = "https://finnhub.io/api/v1"
	requestTimeout   = 8 * time.Second
	maxRetries       = 2
)

var retryableStatus = map[int]bool{408: true, 425: true, 429: true, 500: true, 502: true, 503: true, 504: true}

type FmpProvider struct {
	apiKey        string
	finnhubAPIKey string
	client        *http.Client
}

func NewFmpProvider(apiKey, finnhubAPIKey string) *FmpProvider {
	return &FmpProvider{
		apiKey:        apiKey,
		finnhubAPIKey: finnhubAPIKey,
		client:        &http.Client{Timeout: requestTimeout},
	}
}

func (p *FmpProvider) getJSON(path string, target interface{}) error {
	sep := "&"
	if !strings.Contains(path, "?") {
		sep = "?"
	}
	u := fmt.Sprintf("%s%s%sapikey=%s", fmpBaseURL, path, sep, url.QueryEscape(p.apiKey))

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := p.client.Get(u)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(time.Duration(250*(attempt+1)) * time.Millisecond)
				continue
			}
			return &ProviderError{Provider: "fmp", Message: err.Error(), Retriable: true}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			msg := string(body)
			if msg == "" {
				msg = "Request failed"
			}
			if retryableStatus[resp.StatusCode] && attempt < maxRetries {
				time.Sleep(time.Duration(250*(attempt+1)) * time.Millisecond)
				continue
			}
			return &ProviderError{Provider: "fmp", Message: msg, StatusCode: resp.StatusCode, Retriable: retryableStatus[resp.StatusCode]}
		}

		return json.NewDecoder(resp.Body).Decode(target)
	}
	return &ProviderError{Provider: "fmp", Message: "Request failed after retries", Retriable: true}
}

func (p *FmpProvider) getFinnhubJSON(path string, target interface{}) error {
	if p.finnhubAPIKey == "" {
		return fmt.Errorf("finnhub API key not configured")
	}
	sep := "&"
	if !strings.Contains(path, "?") {
		sep = "?"
	}
	u := fmt.Sprintf("%s%s%stoken=%s", finnhubBaseURL, path, sep, url.QueryEscape(p.finnhubAPIKey))
	resp, err := p.client.Get(u)
	if err != nil {
		return &ProviderError{Provider: "finnhub", Message: err.Error(), Retriable: true}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &ProviderError{Provider: "finnhub", Message: string(body), StatusCode: resp.StatusCode, Retriable: retryableStatus[resp.StatusCode]}
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

type fmpSearchRow struct {
	Symbol           string `json:"symbol"`
	Name             string `json:"name"`
	Exchange         string `json:"exchange"`
	ExchangeFullName string `json:"exchangeFullName"`
}

type fmpQuoteRow struct {
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Exchange string  `json:"exchange"`
	Price    float64 `json:"price"`
}

type fmpIncomeRow struct {
	FiscalYear           string  `json:"fiscalYear"`
	Date                 string  `json:"date"`
	EPS                  float64 `json:"eps"`
	Revenue              float64 `json:"revenue"`
	OperatingIncome      float64 `json:"operatingIncome"`
	EBITDA               float64 `json:"ebitda"`
	WeightedAverageShsOut float64 `json:"weightedAverageShsOut"`
}

type fmpCashRow struct {
	FiscalYear         string  `json:"fiscalYear"`
	Date               string  `json:"date"`
	FreeCashFlow       float64 `json:"freeCashFlow"`
	CommonDividendsPaid float64 `json:"commonDividendsPaid"`
}

type fmpBalanceRow struct {
	FiscalYear               string  `json:"fiscalYear"`
	Date                     string  `json:"date"`
	TotalStockholdersEquity  float64 `json:"totalStockholdersEquity"`
	TotalEquity              float64 `json:"totalEquity"`
}

type fmpPricePoint struct {
	Date  string  `json:"date"`
	Close float64 `json:"close"`
}

func parseYear(fiscalYear, date string) int {
	if fiscalYear != "" {
		var y int
		if _, err := fmt.Sscanf(fiscalYear, "%d", &y); err == nil && y > 0 {
			return y
		}
	}
	if len(date) >= 4 {
		var y int
		if _, err := fmt.Sscanf(date[:4], "%d", &y); err == nil && y > 0 {
			return y
		}
	}
	return 0
}

func (p *FmpProvider) SearchTickers(query string) ([]TickerSearchResult, error) {
	normalized := strings.TrimSpace(query)
	if normalized == "" {
		return []TickerSearchResult{}, nil
	}
	var rows []fmpSearchRow
	if err := p.getJSON(fmt.Sprintf("/search-symbol?query=%s&limit=10", url.QueryEscape(normalized)), &rows); err != nil {
		return nil, err
	}
	results := make([]TickerSearchResult, 0, len(rows))
	for _, r := range rows {
		if r.Symbol == "" || r.Name == "" {
			continue
		}
		exchange := r.Exchange
		if exchange == "" {
			exchange = r.ExchangeFullName
		}
		if exchange == "" {
			exchange = "UNKNOWN"
		}
		results = append(results, TickerSearchResult{Symbol: r.Symbol, Name: r.Name, Exchange: exchange})
	}
	return results, nil
}

func (p *FmpProvider) GetTickerSummary(symbol string) (*TickerSummary, error) {
	normalized := strings.ToUpper(symbol)
	enc := url.QueryEscape(normalized)

	var quoteRows []fmpQuoteRow
	var incomeRows []fmpIncomeRow
	var cashRows []fmpCashRow
	var balanceRows []fmpBalanceRow

	type result struct {
		idx int
		err error
	}
	ch := make(chan result, 4)

	go func() { ch <- result{0, p.getJSON(fmt.Sprintf("/quote?symbol=%s", enc), &quoteRows)} }()
	go func() { ch <- result{1, p.getJSON(fmt.Sprintf("/income-statement?symbol=%s&period=annual&limit=1", enc), &incomeRows)} }()
	go func() { ch <- result{2, p.getJSON(fmt.Sprintf("/cash-flow-statement?symbol=%s&period=annual&limit=1", enc), &cashRows)} }()
	go func() { ch <- result{3, p.getJSON(fmt.Sprintf("/balance-sheet-statement?symbol=%s&period=annual&limit=1", enc), &balanceRows)} }()

	for i := 0; i < 4; i++ {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		_ = r.idx
	}

	if len(quoteRows) == 0 || quoteRows[0].Symbol == "" {
		return nil, nil
	}
	q := quoteRows[0]

	var latestIncome fmpIncomeRow
	if len(incomeRows) > 0 {
		latestIncome = incomeRows[0]
	}
	var latestCash fmpCashRow
	if len(cashRows) > 0 {
		latestCash = cashRows[0]
	}
	var latestBalance fmpBalanceRow
	if len(balanceRows) > 0 {
		latestBalance = balanceRows[0]
	}

	shares := latestIncome.WeightedAverageShsOut
	fcfPerShare := 0.0
	if shares > 0 {
		fcfPerShare = latestCash.FreeCashFlow / shares
	}
	bookEquity := latestBalance.TotalStockholdersEquity
	if bookEquity == 0 {
		bookEquity = latestBalance.TotalEquity
	}
	bookPerShare := 0.0
	if shares > 0 {
		bookPerShare = bookEquity / shares
	}
	annualDivPS := 0.0
	if shares > 0 {
		annualDivPS = math.Abs(latestCash.CommonDividendsPaid) / shares
	}

	epsVal := latestIncome.EPS
	if epsVal == 0 {
		epsVal = 1
	}
	fcfVal := fcfPerShare
	if fcfVal <= 0 {
		fcfVal = epsVal
	}

	name := q.Name
	if name == "" {
		name = normalized
	}
	exchange := q.Exchange
	if exchange == "" {
		exchange = "UNKNOWN"
	}

	return &TickerSummary{
		Symbol:   q.Symbol,
		Name:     name,
		Exchange: exchange,
		Currency: "USD",
		Price:    q.Price,
		ValuationBases: ValuationBases{
			EPS:      epsVal,
			FCF:      fcfVal,
			Dividend: annualDivPS,
		},
		TangibleBookValuePerShare: math.Max(bookPerShare, 0),
	}, nil
}

func (p *FmpProvider) GetTickerHistorical(symbol string) (*valuation.TickerHistoricalData, error) {
	normalized := strings.ToUpper(symbol)
	enc := url.QueryEscape(normalized)

	var incomeRows []fmpIncomeRow
	var cashRows []fmpCashRow
	var balanceRows []fmpBalanceRow
	var priceRows []fmpPricePoint

	type result struct {
		idx int
		err error
	}
	ch := make(chan result, 4)

	go func() { ch <- result{0, p.getJSON(fmt.Sprintf("/income-statement?symbol=%s&period=annual&limit=5", enc), &incomeRows)} }()
	go func() { ch <- result{1, p.getJSON(fmt.Sprintf("/cash-flow-statement?symbol=%s&period=annual&limit=5", enc), &cashRows)} }()
	go func() { ch <- result{2, p.getJSON(fmt.Sprintf("/balance-sheet-statement?symbol=%s&period=annual&limit=5", enc), &balanceRows)} }()
	go func() { ch <- result{3, p.getJSON(fmt.Sprintf("/historical-price-eod/full?symbol=%s", enc), &priceRows)} }()

	for i := 0; i < 4; i++ {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
	}

	eps := map[int]float64{}
	revenue := map[int]float64{}
	operatingIncome := map[int]float64{}
	ebitda := map[int]float64{}
	sharesByYear := map[int]float64{}

	for _, row := range incomeRows {
		year := parseYear(row.FiscalYear, row.Date)
		if year == 0 {
			continue
		}
		eps[year] = row.EPS
		revenue[year] = row.Revenue
		operatingIncome[year] = row.OperatingIncome
		ebitda[year] = row.EBITDA
		sharesByYear[year] = row.WeightedAverageShsOut
	}

	fcf := map[int]float64{}
	dividend := map[int]float64{}
	for _, row := range cashRows {
		year := parseYear(row.FiscalYear, row.Date)
		if year == 0 {
			continue
		}
		shares := sharesByYear[year]
		if shares > 0 {
			fcf[year] = row.FreeCashFlow / shares
			dividend[year] = math.Abs(row.CommonDividendsPaid) / shares
		}
	}

	bookValue := map[int]float64{}
	for _, row := range balanceRows {
		year := parseYear(row.FiscalYear, row.Date)
		if year == 0 {
			continue
		}
		equity := row.TotalStockholdersEquity
		if equity == 0 {
			equity = row.TotalEquity
		}
		shares := sharesByYear[year]
		if shares > 0 {
			bookValue[year] = equity / shares
		}
	}

	price := map[int]float64{}
	for _, pt := range priceRows {
		if len(pt.Date) < 4 {
			continue
		}
		year := parseYear("", pt.Date)
		if year == 0 {
			continue
		}
		if _, exists := price[year]; !exists {
			price[year] = pt.Close
		}
	}

	diag := &valuation.HistoricalDiagnostics{SupplementStatus: "none"}

	if p.finnhubAPIKey != "" {
		allMaps := []map[int]float64{eps, revenue, operatingIncome, ebitda, fcf, dividend, bookValue}
		before := countMapPoints(allMaps)

		supplement, err := p.getFinnhubHistoricalSupplement(normalized)
		if err == nil {
			mergeMissing(eps, supplement.eps)
			mergeMissing(revenue, supplement.revenue)
			mergeMissing(operatingIncome, supplement.operatingIncome)
			mergeMissing(ebitda, supplement.ebitda)
			mergeMissing(fcf, supplement.fcf)
			mergeMissing(dividend, supplement.dividend)
			mergeMissing(bookValue, supplement.bookValue)

			after := countMapPoints(allMaps)
			filled := after - before
			if filled < 0 {
				filled = 0
			}
			status := "finnhub-no-fill"
			if filled > 0 {
				status = "finnhub-applied"
			}
			diag = &valuation.HistoricalDiagnostics{SupplementStatus: status, FilledPoints: filled}
		} else {
			diag = &valuation.HistoricalDiagnostics{SupplementStatus: "finnhub-failed"}
		}
	}

	if len(eps) == 0 {
		return nil, nil
	}

	return &valuation.TickerHistoricalData{
		EPS:             toSeries(eps),
		Revenue:         toSeries(revenue),
		OperatingIncome: toSeries(operatingIncome),
		EBITDA:          toSeries(ebitda),
		FCF:             toSeries(fcf),
		Dividend:        toSeries(dividend),
		BookValue:       toSeries(bookValue),
		Price:           toSeries(price),
		Diagnostics:     diag,
	}, nil
}

type finnhubStatementItem struct {
	Concept string      `json:"concept"`
	Value   interface{} `json:"value"`
}

type finnhubReport struct {
	IC []finnhubStatementItem `json:"ic"`
	CF []finnhubStatementItem `json:"cf"`
	BS []finnhubStatementItem `json:"bs"`
}

type finnhubFinancialReport struct {
	Year    interface{}    `json:"year"`
	EndDate string         `json:"endDate"`
	Report  *finnhubReport `json:"report"`
}

type finnhubResponse struct {
	Data []finnhubFinancialReport `json:"data"`
}

type finnhubSupplement struct {
	eps             map[int]float64
	revenue         map[int]float64
	operatingIncome map[int]float64
	ebitda          map[int]float64
	fcf             map[int]float64
	dividend        map[int]float64
	bookValue       map[int]float64
}

func (p *FmpProvider) getFinnhubHistoricalSupplement(symbol string) (*finnhubSupplement, error) {
	var resp finnhubResponse
	if err := p.getFinnhubJSON(fmt.Sprintf("/stock/financials-reported?symbol=%s", url.QueryEscape(symbol)), &resp); err != nil {
		return nil, err
	}

	sup := &finnhubSupplement{
		eps: map[int]float64{}, revenue: map[int]float64{}, operatingIncome: map[int]float64{},
		ebitda: map[int]float64{}, fcf: map[int]float64{}, dividend: map[int]float64{}, bookValue: map[int]float64{},
	}

	for _, report := range resp.Data {
		year := parseFinnhubYear(report)
		if year == 0 {
			continue
		}
		var ic, cf, bs []finnhubStatementItem
		if report.Report != nil {
			ic = report.Report.IC
			cf = report.Report.CF
			bs = report.Report.BS
		}

		shares := statementValue(ic, []string{
			"us-gaap_WeightedAverageNumberOfDilutedSharesOutstanding",
			"us-gaap_WeightedAverageNumberOfShareOutstandingBasicAndDiluted",
			"us-gaap_WeightedAverageNumberOfSharesOutstandingBasic",
			"dei_EntityCommonStockSharesOutstanding",
		})
		revenueVal := statementValue(ic, []string{
			"us-gaap_Revenues",
			"us-gaap_RevenueFromContractWithCustomerExcludingAssessedTax",
			"us-gaap_SalesRevenueNet",
		})
		opIncomeVal := statementValue(ic, []string{"us-gaap_OperatingIncomeLoss"})
		epsVal := statementValue(ic, []string{"us-gaap_EarningsPerShareDiluted", "us-gaap_EarningsPerShareBasic"})
		depreciationVal := statementValue(cf, []string{
			"us-gaap_DepreciationDepletionAndAmortization",
			"us-gaap_DepreciationAmortizationAndAccretionNet",
		})
		opCashFlowVal := statementValue(cf, []string{"us-gaap_NetCashProvidedByUsedInOperatingActivities"})
		capexRaw := statementValue(cf, []string{
			"us-gaap_PaymentsToAcquirePropertyPlantAndEquipment",
			"us-gaap_PaymentsToAcquireProductiveAssets",
		})
		capex := math.Abs(capexRaw)
		divPaymentsRaw := statementValue(cf, []string{"us-gaap_PaymentsOfDividends", "us-gaap_PaymentsOfOrdinaryDividends"})
		divPSDirect := statementValue(ic, []string{
			"us-gaap_CommonStockDividendsPerShareDeclared",
			"us-gaap_CommonStockDividendsPerShareCashPaid",
		})
		equityVal := statementValue(bs, []string{
			"us-gaap_StockholdersEquity",
			"us-gaap_StockholdersEquityIncludingPortionAttributableToNoncontrollingInterest",
		})

		if epsVal > 0 {
			sup.eps[year] = epsVal
		}
		if revenueVal > 0 {
			sup.revenue[year] = revenueVal
		}
		if opIncomeVal > 0 {
			sup.operatingIncome[year] = opIncomeVal
		}
		if opIncomeVal > 0 && depreciationVal != 0 {
			sup.ebitda[year] = opIncomeVal + math.Abs(depreciationVal)
		}
		if opCashFlowVal != 0 && shares > 0 {
			sup.fcf[year] = (opCashFlowVal - capex) / shares
		}
		if divPSDirect > 0 {
			sup.dividend[year] = divPSDirect
		} else if divPaymentsRaw != 0 && shares > 0 {
			sup.dividend[year] = math.Abs(divPaymentsRaw) / shares
		}
		if equityVal > 0 && shares > 0 {
			sup.bookValue[year] = equityVal / shares
		}
	}

	return sup, nil
}

func parseFinnhubYear(r finnhubFinancialReport) int {
	switch v := r.Year.(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case string:
		var y int
		if _, err := fmt.Sscanf(v, "%d", &y); err == nil && y > 0 {
			return y
		}
	}
	if len(r.EndDate) >= 4 {
		var y int
		if _, err := fmt.Sscanf(r.EndDate[:4], "%d", &y); err == nil && y > 0 {
			return y
		}
	}
	return 0
}

func statementValue(items []finnhubStatementItem, concepts []string) float64 {
	for _, concept := range concepts {
		for _, item := range items {
			if item.Concept == concept {
				switch v := item.Value.(type) {
				case float64:
					if v != 0 {
						return v
					}
				case string:
					var f float64
					if _, err := fmt.Sscanf(v, "%f", &f); err == nil && f != 0 {
						return f
					}
				}
			}
		}
	}
	return 0
}

func toSeries(m map[int]float64) valuation.HistoricalSeries {
	years := make([]int, 0, len(m))
	for y := range m {
		years = append(years, y)
	}
	sort.Ints(years)
	values := make([]float64, len(years))
	for i, y := range years {
		values[i] = m[y]
	}
	return valuation.HistoricalSeries{Years: years, Values: values}
}

func mergeMissing(target, source map[int]float64) {
	for k, v := range source {
		if _, exists := target[k]; !exists {
			target[k] = v
		}
	}
}

func countMapPoints(maps []map[int]float64) int {
	total := 0
	for _, m := range maps {
		total += len(m)
	}
	return total
}
