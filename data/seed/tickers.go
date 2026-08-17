package seed

import (
	"stock-advisor-site-golang/valuation"
	"strings"
)

type TickerSeed struct {
	Symbol                    string
	Name                      string
	Exchange                  string
	Currency                  string
	Price                     float64
	EPS                       float64
	FCFPerShare               float64
	DividendPerShare          float64
	TangibleBookValuePerShare float64
	Historical                valuation.TickerHistoricalData
}

var Seeds = []TickerSeed{
	{
		Symbol:                    "NVDA",
		Name:                      "NVIDIA Corp",
		Exchange:                  "NASDAQ",
		Currency:                  "USD",
		Price:                     184.47,
		EPS:                       5.35,
		FCFPerShare:               4.9,
		DividendPerShare:          0.04,
		TangibleBookValuePerShare: 5.48,
		Historical: valuation.TickerHistoricalData{
			EPS:             valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{0.25, 0.3, 0.36, 0.46, 0.6, 0.72, 0.9, 1.25, 1.8, 3.4, 5.35}},
			Revenue:         valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{1.5, 1.6, 1.8, 2.0, 2.2, 2.35, 2.7, 3.1, 3.6, 4.2, 4.9}},
			OperatingIncome: valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{0.22, 0.25, 0.29, 0.35, 0.42, 0.46, 0.53, 0.71, 0.96, 1.65, 2.45}},
			EBITDA:          valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{0.28, 0.31, 0.36, 0.44, 0.52, 0.58, 0.66, 0.86, 1.12, 1.9, 2.8}},
			FCF:             valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{0.2, 0.22, 0.26, 0.31, 0.39, 0.43, 0.5, 0.67, 0.95, 1.8, 2.6}},
			Dividend:        valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{0.02, 0.02, 0.02, 0.025, 0.03, 0.03, 0.03, 0.035, 0.035, 0.04, 0.04}},
			BookValue:       valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{1.2, 1.3, 1.45, 1.62, 1.8, 2.0, 2.3, 2.7, 3.15, 4.1, 5.48}},
			Price:           valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{24, 28, 35, 43, 50, 57, 74, 95, 121, 150, 184.47}},
		},
	},
	{
		Symbol:                    "AAPL",
		Name:                      "Apple Inc",
		Exchange:                  "NASDAQ",
		Currency:                  "USD",
		Price:                     226.52,
		EPS:                       6.7,
		FCFPerShare:               7.22,
		DividendPerShare:          0.99,
		TangibleBookValuePerShare: 4.21,
		Historical: valuation.TickerHistoricalData{
			EPS:             valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{2.3, 2.1, 2.4, 2.9, 3.0, 3.3, 4.1, 5.2, 5.8, 6.2, 6.7}},
			Revenue:         valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{8.4, 8.1, 8.6, 9.0, 9.3, 9.6, 10.1, 10.8, 11.2, 11.7, 12.2}},
			OperatingIncome: valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{1.9, 1.85, 2.0, 2.2, 2.3, 2.4, 2.7, 3.1, 3.3, 3.45, 3.6}},
			EBITDA:          valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{2.1, 2.05, 2.2, 2.45, 2.6, 2.75, 3.0, 3.4, 3.6, 3.85, 4.1}},
			FCF:             valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{2.4, 2.3, 2.6, 2.9, 3.0, 3.2, 3.55, 4.1, 4.6, 5.3, 7.22}},
			Dividend:        valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{0.47, 0.5, 0.55, 0.62, 0.72, 0.77, 0.82, 0.88, 0.92, 0.96, 0.99}},
			BookValue:       valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{3.4, 3.2, 3.3, 3.45, 3.55, 3.7, 3.82, 3.95, 4.02, 4.1, 4.21}},
			Price:           valuation.HistoricalSeries{Years: []int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025}, Values: []float64{26, 28, 31, 38, 44, 59, 76, 112, 147, 189, 226.52}},
		},
	},
}

func FindBySymbol(symbol string) *TickerSeed {
	upper := strings.ToUpper(symbol)
	for i := range Seeds {
		if Seeds[i].Symbol == upper {
			return &Seeds[i]
		}
	}
	return nil
}

func SearchSeeds(query string) []TickerSeed {
	upper := strings.ToUpper(strings.TrimSpace(query))
	if upper == "" {
		return nil
	}
	var results []TickerSeed
	for _, s := range Seeds {
		if strings.Contains(s.Symbol, upper) || strings.Contains(strings.ToUpper(s.Name), upper) {
			results = append(results, s)
		}
		if len(results) >= 10 {
			break
		}
	}
	return results
}
