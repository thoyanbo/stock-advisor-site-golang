package valuation

import "math"

type HistoricalSeries struct {
	Years  []int     `json:"years"`
	Values []float64 `json:"values"`
}

type HistoricalDiagnostics struct {
	SupplementStatus string `json:"supplementStatus"`
	FilledPoints     int    `json:"filledPoints,omitempty"`
}

type TickerHistoricalData struct {
	EPS             HistoricalSeries       `json:"eps"`
	Revenue         HistoricalSeries       `json:"revenue"`
	OperatingIncome HistoricalSeries       `json:"operatingIncome"`
	EBITDA          HistoricalSeries       `json:"ebitda"`
	FCF             HistoricalSeries       `json:"fcf"`
	Dividend        HistoricalSeries       `json:"dividend"`
	BookValue       HistoricalSeries       `json:"bookValue"`
	Price           HistoricalSeries       `json:"price"`
	Diagnostics     *HistoricalDiagnostics `json:"diagnostics,omitempty"`
}

type HistoricalGrowthRow struct {
	Metric string              `json:"metric"`
	Values map[string]*float64 `json:"values"`
}

type GrowthHorizon struct {
	Label string
	Years int
}

var Horizons = []GrowthHorizon{
	{"1Y", 1}, {"3Y", 3}, {"5Y", 5}, {"10Y", 10},
}

func ComputeCagr(series HistoricalSeries, years int) *float64 {
	if len(series.Years) != len(series.Values) || len(series.Values) == 0 {
		return nil
	}
	latestIndex := len(series.Values) - 1
	latestYear := series.Years[latestIndex]
	startYear := latestYear - years

	startIndex := -1
	for i, y := range series.Years {
		if y == startYear {
			startIndex = i
			break
		}
	}
	if startIndex < 0 {
		return nil
	}

	start := series.Values[startIndex]
	end := series.Values[latestIndex]
	if start <= 0 || end <= 0 {
		return nil
	}

	cagr := (math.Pow(end/start, 1/float64(years)) - 1) * 100
	v := round2(cagr)
	return &v
}

func BuildHistoricalGrowthRows(data *TickerHistoricalData) []HistoricalGrowthRow {
	type metricDef struct {
		key    string
		series HistoricalSeries
	}
	metrics := []metricDef{
		{"eps", data.EPS},
		{"revenue", data.Revenue},
		{"operatingIncome", data.OperatingIncome},
		{"ebitda", data.EBITDA},
		{"fcf", data.FCF},
		{"dividend", data.Dividend},
		{"bookValue", data.BookValue},
		{"price", data.Price},
	}

	rows := make([]HistoricalGrowthRow, 0, len(metrics))
	for _, m := range metrics {
		values := map[string]*float64{}
		for _, h := range Horizons {
			v := ComputeCagr(m.series, h.Years)
			if v != nil {
				values[h.Label] = v
			}
		}
		rows = append(rows, HistoricalGrowthRow{Metric: m.key, Values: values})
	}
	return rows
}

type DataQualityAssessment struct {
	Score       float64 `json:"score"`
	CoveragePct float64 `json:"coveragePct"`
	Level       string  `json:"level"`
	Message     string  `json:"message"`
}

func AssessHistoricalDataQuality(rows []HistoricalGrowthRow) DataQualityAssessment {
	horizons := []string{"1Y", "3Y", "5Y", "10Y"}
	totalCells := len(rows) * len(horizons)
	populated := 0
	for _, row := range rows {
		for _, h := range horizons {
			if row.Values[h] != nil {
				populated++
			}
		}
	}
	coverageRatio := 0.0
	if totalCells > 0 {
		coverageRatio = float64(populated) / float64(totalCells)
	}
	coveragePct := round2(coverageRatio * 100)
	score := coveragePct
	if score > 100 {
		score = 100
	}

	level := "low"
	if coveragePct >= 75 {
		level = "high"
	} else if coveragePct >= 40 {
		level = "medium"
	}

	message := "Historical coverage is sparse; treat trend-based valuation signals cautiously."
	if level == "high" {
		message = "Historical coverage is strong; confidence in trend comparisons is high."
	} else if level == "medium" {
		message = "Historical coverage is partial; interpret valuation trends with moderate confidence."
	}

	return DataQualityAssessment{Score: score, CoveragePct: coveragePct, Level: level, Message: message}
}

var HorizonPreference = []string{"10Y", "5Y", "3Y", "1Y"}

func DeriveGrowthRateFromRows(basisType string, rows []HistoricalGrowthRow, fallbackPct float64) float64 {
	primaryMetric := "eps"
	if basisType == "fcf" {
		primaryMetric = "fcf"
	} else if basisType == "dividend" {
		primaryMetric = "dividend"
	}

	if v := selectHistoricalGrowth(rows, primaryMetric); v != nil {
		return clampGrowth(*v)
	}
	if v := selectHistoricalGrowth(rows, "eps"); v != nil {
		return clampGrowth(*v)
	}
	return clampGrowth(fallbackPct)
}

func selectHistoricalGrowth(rows []HistoricalGrowthRow, metric string) *float64 {
	for _, row := range rows {
		if row.Metric == metric {
			for _, h := range HorizonPreference {
				if v, ok := row.Values[h]; ok && v != nil {
					return v
				}
			}
		}
	}
	return nil
}

func clampGrowth(v float64) float64 {
	if v < 5 {
		return 5
	}
	if v > 20 {
		return 20
	}
	return round2(v)
}
