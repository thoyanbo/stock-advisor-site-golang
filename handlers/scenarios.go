package handlers

import (
	"encoding/json"
	"net/http"
	"stock-advisor-site-golang/data/seed"
	"stock-advisor-site-golang/repository"
	"strings"

	"github.com/go-chi/chi/v5"
)

type ScenarioHandlers struct {
	repo *repository.ScenarioRepository
}

func NewScenarioHandlers(repo *repository.ScenarioRepository) *ScenarioHandlers {
	return &ScenarioHandlers{repo: repo}
}

func (h *ScenarioHandlers) List(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	if seed.FindBySymbol(symbol) == nil {
		writeError(w, http.StatusNotFound, "Ticker not found.")
		return
	}
	scenarios, err := h.repo.List(symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list scenarios.")
		return
	}
	if scenarios == nil {
		scenarios = []repository.ScenarioRecord{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"scenarios": scenarios})
}

func (h *ScenarioHandlers) Create(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	if seed.FindBySymbol(symbol) == nil {
		writeError(w, http.StatusNotFound, "Ticker not found.")
		return
	}

	var input repository.ScenarioCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid scenario payload.")
		return
	}

	if errs := validateScenarioInput(input); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid scenario payload.",
			"details": errs,
		})
		return
	}

	record, err := h.repo.Create(symbol, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create scenario.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"scenario": record})
}

func (h *ScenarioHandlers) Update(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	scenarioID := chi.URLParam(r, "scenarioId")

	var input repository.ScenarioUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid scenario payload.")
		return
	}

	if errs := validateScenarioInput(input); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid scenario payload.",
			"details": errs,
		})
		return
	}

	record, err := h.repo.Update(symbol, scenarioID, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update scenario.")
		return
	}
	if record == nil {
		writeError(w, http.StatusNotFound, "Scenario not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"scenario": record})
}

func (h *ScenarioHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	scenarioID := chi.URLParam(r, "scenarioId")

	deleted, err := h.repo.Delete(symbol, scenarioID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete scenario.")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "Scenario not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": true})
}

func validateScenarioInput(input repository.ScenarioCreateInput) []validationError {
	var errs []validationError
	name := strings.TrimSpace(input.Name)
	if len(name) < 1 || len(name) > 80 {
		errs = append(errs, validationError{"name", "Must be 1–80 characters"})
	}
	if input.BasisType != "eps" && input.BasisType != "fcf" && input.BasisType != "dividend" {
		errs = append(errs, validationError{"basisType", "Must be eps, fcf, or dividend"})
	}
	if input.DiscountRatePct <= 0 || input.DiscountRatePct > 50 {
		errs = append(errs, validationError{"discountRatePct", "Must be > 0 and <= 50"})
	}
	if input.GrowthYears < 1 || input.GrowthYears > 30 {
		errs = append(errs, validationError{"growthYears", "Must be 1–30"})
	}
	if input.GrowthRatePct < -50 || input.GrowthRatePct > 100 {
		errs = append(errs, validationError{"growthRatePct", "Must be -50 to 100"})
	}
	if input.TerminalYears < 1 || input.TerminalYears > 30 {
		errs = append(errs, validationError{"terminalYears", "Must be 1–30"})
	}
	if input.TerminalRatePct < -50 || input.TerminalRatePct > 100 {
		errs = append(errs, validationError{"terminalRatePct", "Must be -50 to 100"})
	}
	if input.TerminalRatePct >= input.DiscountRatePct {
		errs = append(errs, validationError{"terminalRatePct", "Must be less than discountRatePct"})
	}
	return errs
}
