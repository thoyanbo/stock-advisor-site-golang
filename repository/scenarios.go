package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ScenarioRecord struct {
	ID              string  `json:"id"`
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	BasisType       string  `json:"basisType"`
	DiscountRatePct float64 `json:"discountRatePct"`
	GrowthYears     int     `json:"growthYears"`
	GrowthRatePct   float64 `json:"growthRatePct"`
	TerminalYears   int     `json:"terminalYears"`
	TerminalRatePct float64 `json:"terminalRatePct"`
	AddTangibleBook bool    `json:"addTangibleBook"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type ScenarioCreateInput struct {
	Name            string  `json:"name"`
	BasisType       string  `json:"basisType"`
	DiscountRatePct float64 `json:"discountRatePct"`
	GrowthYears     int     `json:"growthYears"`
	GrowthRatePct   float64 `json:"growthRatePct"`
	TerminalYears   int     `json:"terminalYears"`
	TerminalRatePct float64 `json:"terminalRatePct"`
	AddTangibleBook bool    `json:"addTangibleBook"`
}

type ScenarioUpdateInput = ScenarioCreateInput

type ScenarioRepository struct {
	storagePath string
	mu          sync.Mutex
}

func NewScenarioRepository(storagePath string) *ScenarioRepository {
	if storagePath == "" {
		storagePath = "data/local/scenarios.json"
	}
	return &ScenarioRepository{storagePath: storagePath}
}

func (r *ScenarioRepository) readAll() ([]ScenarioRecord, error) {
	data, err := os.ReadFile(r.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScenarioRecord{}, nil
		}
		return nil, err
	}
	var records []ScenarioRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return []ScenarioRecord{}, nil
	}
	return records, nil
}

func (r *ScenarioRepository) writeAll(records []ScenarioRecord) error {
	dir := filepath.Dir(r.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.storagePath, data, 0644)
}

func (r *ScenarioRepository) List(symbol string) ([]ScenarioRecord, error) {
	records, err := r.readAll()
	if err != nil {
		return nil, err
	}
	upper := symbol
	var filtered []ScenarioRecord
	for _, rec := range records {
		if rec.Symbol == upper {
			filtered = append(filtered, rec)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt > filtered[j].UpdatedAt
	})
	return filtered, nil
}

func (r *ScenarioRepository) Create(symbol string, input ScenarioCreateInput) (*ScenarioRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	records, err := r.readAll()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	record := ScenarioRecord{
		ID:              uuid.New().String(),
		Symbol:          symbol,
		Name:            input.Name,
		BasisType:       input.BasisType,
		DiscountRatePct: input.DiscountRatePct,
		GrowthYears:     input.GrowthYears,
		GrowthRatePct:   input.GrowthRatePct,
		TerminalYears:   input.TerminalYears,
		TerminalRatePct: input.TerminalRatePct,
		AddTangibleBook: input.AddTangibleBook,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := r.writeAll(records); err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *ScenarioRepository) Update(symbol, scenarioID string, input ScenarioUpdateInput) (*ScenarioRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	records, err := r.readAll()
	if err != nil {
		return nil, err
	}

	for i, rec := range records {
		if rec.ID == scenarioID && rec.Symbol == symbol {
			records[i].Name = input.Name
			records[i].BasisType = input.BasisType
			records[i].DiscountRatePct = input.DiscountRatePct
			records[i].GrowthYears = input.GrowthYears
			records[i].GrowthRatePct = input.GrowthRatePct
			records[i].TerminalYears = input.TerminalYears
			records[i].TerminalRatePct = input.TerminalRatePct
			records[i].AddTangibleBook = input.AddTangibleBook
			records[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := r.writeAll(records); err != nil {
				return nil, err
			}
			return &records[i], nil
		}
	}
	return nil, nil
}

func (r *ScenarioRepository) Delete(symbol, scenarioID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	records, err := r.readAll()
	if err != nil {
		return false, err
	}

	var filtered []ScenarioRecord
	found := false
	for _, rec := range records {
		if rec.ID == scenarioID && rec.Symbol == symbol {
			found = true
			continue
		}
		filtered = append(filtered, rec)
	}
	if !found {
		return false, nil
	}
	if err := r.writeAll(filtered); err != nil {
		return false, err
	}
	return true, nil
}
