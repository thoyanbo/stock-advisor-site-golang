package valuation

import "math"

type DCFInput struct {
	BasisType                 string  `json:"basisType"`
	CurrentPrice              float64 `json:"currentPrice"`
	BaseValuePerShare         float64 `json:"baseValuePerShare"`
	DiscountRatePct           float64 `json:"discountRatePct"`
	GrowthYears               int     `json:"growthYears"`
	GrowthRatePct             float64 `json:"growthRatePct"`
	TerminalYears             int     `json:"terminalYears"`
	TerminalRatePct           float64 `json:"terminalRatePct"`
	AddTangibleBook           bool    `json:"addTangibleBook"`
	TangibleBookValuePerShare float64 `json:"tangibleBookValuePerShare"`
}

type DCFOutput struct {
	FairValue                 float64 `json:"fairValue"`
	MarginOfSafetyPct         float64 `json:"marginOfSafetyPct"`
	PresentValueGrowthStage   float64 `json:"presentValueGrowthStage"`
	PresentValueTerminalStage float64 `json:"presentValueTerminalStage"`
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func ComputeDCF(input DCFInput) DCFOutput {
	discount := input.DiscountRatePct / 100
	growth := input.GrowthRatePct / 100
	terminalGrowth := input.TerminalRatePct / 100

	pvGrowthStage := 0.0
	projected := input.BaseValuePerShare

	for year := 1; year <= input.GrowthYears; year++ {
		projected *= 1 + growth
		pvGrowthStage += projected / math.Pow(1+discount, float64(year))
	}

	pvTerminalStage := 0.0
	firstTerminalYearBase := projected
	for year := 1; year <= input.TerminalYears; year++ {
		terminalProjected := firstTerminalYearBase * math.Pow(1+terminalGrowth, float64(year))
		discounted := terminalProjected / math.Pow(1+discount, float64(input.GrowthYears+year))
		pvTerminalStage += discounted
	}

	fairValue := pvGrowthStage + pvTerminalStage
	if input.AddTangibleBook {
		fairValue += input.TangibleBookValuePerShare
	}

	marginOfSafetyPct := ((fairValue - input.CurrentPrice) / input.CurrentPrice) * 100

	return DCFOutput{
		FairValue:                 round2(fairValue),
		MarginOfSafetyPct:         round2(marginOfSafetyPct),
		PresentValueGrowthStage:   round2(pvGrowthStage),
		PresentValueTerminalStage: round2(pvTerminalStage),
	}
}
