package valuation

import "math"

type ReverseDCFInput struct {
	CurrentPrice              float64 `json:"currentPrice"`
	BaseValuePerShare         float64 `json:"baseValuePerShare"`
	DiscountRatePct           float64 `json:"discountRatePct"`
	GrowthYears               int     `json:"growthYears"`
	TerminalYears             int     `json:"terminalYears"`
	TerminalRatePct           float64 `json:"terminalRatePct"`
	AddTangibleBook           bool    `json:"addTangibleBook"`
	TangibleBookValuePerShare float64 `json:"tangibleBookValuePerShare"`
}

type ReverseDCFOutput struct {
	ImpliedGrowthRatePct      float64 `json:"impliedGrowthRatePct"`
	ModelPriceAtImpliedGrowth float64 `json:"modelPriceAtImpliedGrowth"`
	Converged                 bool    `json:"converged"`
	Iterations                int     `json:"iterations"`
}

func computeModelPrice(input ReverseDCFInput, growthRatePct float64) float64 {
	discount := input.DiscountRatePct / 100
	growth := growthRatePct / 100
	terminalGrowth := input.TerminalRatePct / 100

	projected := input.BaseValuePerShare
	pvGrowth := 0.0
	for year := 1; year <= input.GrowthYears; year++ {
		projected *= 1 + growth
		pvGrowth += projected / math.Pow(1+discount, float64(year))
	}

	pvTerminal := 0.0
	terminalBase := projected
	for year := 1; year <= input.TerminalYears; year++ {
		terminalProjected := terminalBase * math.Pow(1+terminalGrowth, float64(year))
		pvTerminal += terminalProjected / math.Pow(1+discount, float64(input.GrowthYears+year))
	}

	fairValue := pvGrowth + pvTerminal
	if input.AddTangibleBook {
		fairValue += input.TangibleBookValuePerShare
	}
	return fairValue
}

func ComputeReverseDCF(input ReverseDCFInput) ReverseDCFOutput {
	target := input.CurrentPrice
	left := -50.0
	right := 100.0
	bestGrowth := 0.0
	bestPrice := computeModelPrice(input, bestGrowth)
	converged := false
	iterations := 0

	for i := 0; i < 80; i++ {
		iterations++
		mid := (left + right) / 2
		price := computeModelPrice(input, mid)
		bestGrowth = mid
		bestPrice = price

		if math.Abs(price-target) <= 0.01 {
			converged = true
			break
		}

		if price > target {
			right = mid
		} else {
			left = mid
		}
	}

	return ReverseDCFOutput{
		ImpliedGrowthRatePct:      round2(bestGrowth),
		ModelPriceAtImpliedGrowth: round2(bestPrice),
		Converged:                 converged,
		Iterations:                iterations,
	}
}
