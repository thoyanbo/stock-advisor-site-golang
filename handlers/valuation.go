package handlers

import (
	"encoding/json"
	"net/http"
	"stock-advisor-site-golang/valuation"
)

type dcfRequest struct {
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

type reverseDCFRequest struct {
	CurrentPrice              float64 `json:"currentPrice"`
	BaseValuePerShare         float64 `json:"baseValuePerShare"`
	DiscountRatePct           float64 `json:"discountRatePct"`
	GrowthYears               int     `json:"growthYears"`
	TerminalYears             int     `json:"terminalYears"`
	TerminalRatePct           float64 `json:"terminalRatePct"`
	AddTangibleBook           bool    `json:"addTangibleBook"`
	TangibleBookValuePerShare float64 `json:"tangibleBookValuePerShare"`
}

func DCFHandler(w http.ResponseWriter, r *http.Request) {
	var req dcfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid DCF input payload.")
		return
	}

	errs := validateDCFInput(req)
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid DCF input payload.",
			"details": errs,
		})
		return
	}

	result := valuation.ComputeDCF(valuation.DCFInput{
		BasisType:                 req.BasisType,
		CurrentPrice:              req.CurrentPrice,
		BaseValuePerShare:         req.BaseValuePerShare,
		DiscountRatePct:           req.DiscountRatePct,
		GrowthYears:               req.GrowthYears,
		GrowthRatePct:             req.GrowthRatePct,
		TerminalYears:             req.TerminalYears,
		TerminalRatePct:           req.TerminalRatePct,
		AddTangibleBook:           req.AddTangibleBook,
		TangibleBookValuePerShare: req.TangibleBookValuePerShare,
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"result": result})
}

func ReverseDCFHandler(w http.ResponseWriter, r *http.Request) {
	var req reverseDCFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid reverse DCF input payload.")
		return
	}

	errs := validateReverseDCFInput(req)
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid reverse DCF input payload.",
			"details": errs,
		})
		return
	}

	result := valuation.ComputeReverseDCF(valuation.ReverseDCFInput{
		CurrentPrice:              req.CurrentPrice,
		BaseValuePerShare:         req.BaseValuePerShare,
		DiscountRatePct:           req.DiscountRatePct,
		GrowthYears:               req.GrowthYears,
		TerminalYears:             req.TerminalYears,
		TerminalRatePct:           req.TerminalRatePct,
		AddTangibleBook:           req.AddTangibleBook,
		TangibleBookValuePerShare: req.TangibleBookValuePerShare,
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"result": result})
}

type validationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func validateDCFInput(r dcfRequest) []validationError {
	var errs []validationError
	if r.BasisType != "eps" && r.BasisType != "fcf" && r.BasisType != "dividend" {
		errs = append(errs, validationError{"basisType", "Must be eps, fcf, or dividend"})
	}
	if r.CurrentPrice <= 0 {
		errs = append(errs, validationError{"currentPrice", "Must be > 0"})
	}
	if r.BaseValuePerShare <= 0 {
		errs = append(errs, validationError{"baseValuePerShare", "Must be > 0"})
	}
	if r.DiscountRatePct <= 0 || r.DiscountRatePct > 50 {
		errs = append(errs, validationError{"discountRatePct", "Must be > 0 and <= 50"})
	}
	if r.GrowthYears < 1 || r.GrowthYears > 30 {
		errs = append(errs, validationError{"growthYears", "Must be 1–30"})
	}
	if r.GrowthRatePct < -50 || r.GrowthRatePct > 100 {
		errs = append(errs, validationError{"growthRatePct", "Must be -50 to 100"})
	}
	if r.TerminalYears < 1 || r.TerminalYears > 30 {
		errs = append(errs, validationError{"terminalYears", "Must be 1–30"})
	}
	if r.TerminalRatePct < -50 || r.TerminalRatePct > 100 {
		errs = append(errs, validationError{"terminalRatePct", "Must be -50 to 100"})
	}
	if r.TerminalRatePct >= r.DiscountRatePct {
		errs = append(errs, validationError{"terminalRatePct", "Must be less than discountRatePct"})
	}
	return errs
}

func validateReverseDCFInput(r reverseDCFRequest) []validationError {
	var errs []validationError
	if r.CurrentPrice <= 0 {
		errs = append(errs, validationError{"currentPrice", "Must be > 0"})
	}
	if r.BaseValuePerShare <= 0 {
		errs = append(errs, validationError{"baseValuePerShare", "Must be > 0"})
	}
	if r.DiscountRatePct <= 0 || r.DiscountRatePct > 50 {
		errs = append(errs, validationError{"discountRatePct", "Must be > 0 and <= 50"})
	}
	if r.GrowthYears < 1 || r.GrowthYears > 30 {
		errs = append(errs, validationError{"growthYears", "Must be 1–30"})
	}
	if r.TerminalYears < 1 || r.TerminalYears > 30 {
		errs = append(errs, validationError{"terminalYears", "Must be 1–30"})
	}
	if r.TerminalRatePct < -50 || r.TerminalRatePct > 100 {
		errs = append(errs, validationError{"terminalRatePct", "Must be -50 to 100"})
	}
	if r.TerminalRatePct >= r.DiscountRatePct {
		errs = append(errs, validationError{"terminalRatePct", "Must be less than discountRatePct"})
	}
	return errs
}
