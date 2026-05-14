package budget

import (
	"context"
	"fmt"
	"log"
	"strings"

	"warehouse/internal/settings"
)

// SheetReader abstracts Google Sheets access so tests can inject fake data.
type SheetReader interface {
	FetchSheet(spreadsheetID, sheetName string) ([][]string, error)
}

type Service struct {
	repo            *Repository
	sheetReader     SheetReader
	settingsRepo    *settings.Repository
	fallbackSheetID string
}

func NewService(repo *Repository, sheetReader SheetReader, settingsRepo *settings.Repository, fallbackSheetID string) *Service {
	return &Service{
		repo:            repo,
		sheetReader:     sheetReader,
		settingsRepo:    settingsRepo,
		fallbackSheetID: fallbackSheetID,
	}
}

func (s *Service) getSheetID(ctx context.Context) string {
	id, _ := s.settingsRepo.Get(ctx, "equipment_request.sheet_id")
	if id == "" {
		return s.fallbackSheetID
	}
	return id
}

func (s *Service) getCennikSheetName(ctx context.Context) string {
	name, _ := s.settingsRepo.Get(ctx, "equipment_request.cennik_sheet_name")
	if name == "" {
		return "Cennik"
	}
	return name
}

// GetBudgetSummary returns the full dynamic budget summary.
// vatMultiplier: 1.0 for net, 1.23 for gross (z VAT).
func (s *Service) GetBudgetSummary(ctx context.Context, budgetOwner string, vatMultiplier float64) (*BudgetSummary, error) {
	summary, err := s.repo.GetBudgetSummary(ctx, Filter{BudgetOwner: budgetOwner})
	if err != nil {
		return nil, err
	}
	if vatMultiplier != 1.0 {
		for i := range summary.Items {
			for j := range summary.Items[i].Prices {
				summary.Items[i].Prices[j].UnitPrice *= vatMultiplier
				summary.Items[i].Prices[j].Total *= vatMultiplier
			}
		}
		for i := range summary.SupplierTotals {
			summary.SupplierTotals[i].Total *= vatMultiplier
		}
	}
	return summary, nil
}

func (s *Service) GetBudgetPersons(ctx context.Context) ([]string, error) {
	return s.repo.GetBudgetPersons(ctx)
}

func (s *Service) ListPrices(ctx context.Context) ([]PriceListItem, error) {
	return s.repo.ListPrices(ctx)
}

func (s *Service) ListSuppliers(ctx context.Context) ([]string, error) {
	return s.repo.ListSuppliers(ctx)
}

func (s *Service) UpsertPrice(ctx context.Context, req UpsertPriceRequest) error {
	return s.repo.UpsertPrice(ctx, req)
}

func (s *Service) DeletePrice(ctx context.Context, itemName, supplier string) error {
	return s.repo.DeletePrice(ctx, itemName, supplier)
}

// SyncPricesFromSheetCtx is a hook-compatible wrapper for SetPostSyncHook.
func (s *Service) SyncPricesFromSheetCtx(ctx context.Context) error {
	_, err := s.SyncPricesFromSheet(ctx)
	return err
}

// SyncPricesFromSheet reads the Cennik sheet and upserts prices.
//
// Expected column layout (header row defines supplier names):
//   A          B        C        D          ...
//   Rzeczy     Probis   Netland  Oki-event  ...
//   Laptop     140      90       120
//
// Any number of supplier columns is supported — the header row drives the mapping.
// Legacy 2-col layout (A=name, B=price) is also handled, defaulting supplier to "Cennik".
func (s *Service) SyncPricesFromSheet(ctx context.Context) (int, error) {
	sheetID := s.getSheetID(ctx)
	if sheetID == "" {
		return 0, fmt.Errorf("equipment_request.sheet_id not configured")
	}
	cennikName := s.getCennikSheetName(ctx)

	rows, err := s.sheetReader.FetchSheet(sheetID, cennikName)
	if err != nil {
		return 0, fmt.Errorf("reading Cennik sheet: %w", err)
	}
	if len(rows) < 2 {
		return 0, nil
	}

	parsePrice := func(raw string) (float64, bool) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return 0, false
		}
		raw = strings.ReplaceAll(raw, " ", "")
		raw = strings.ReplaceAll(raw, "zł", "")
		raw = strings.ReplaceAll(raw, ",", ".")
		var v float64
		if _, scanErr := fmt.Sscanf(raw, "%f", &v); scanErr != nil {
			return 0, false
		}
		return v, true
	}

	// Parse supplier names from header row (cols 1+)
	header := rows[0]
	supplierCols := make([]string, len(header))
	for i := 1; i < len(header); i++ {
		supplierCols[i] = strings.TrimSpace(header[i])
	}

	// If no named supplier columns, fall back to single-price mode
	singlePriceMode := len(header) < 2 || supplierCols[1] == "" || strings.EqualFold(supplierCols[1], "cena")

	updated := 0
	for _, row := range rows[1:] {
		if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		name := strings.TrimSpace(row[0])

		if singlePriceMode {
			// Legacy: treat col B (or C if B empty) as a single supplier "Cennik"
			priceCol := ""
			if len(row) >= 2 {
				priceCol = row[1]
			}
			if priceCol == "" && len(row) >= 3 {
				priceCol = row[2]
			}
			if v, ok := parsePrice(priceCol); ok {
				if err := s.repo.UpsertPrice(ctx, UpsertPriceRequest{
					ItemName: name, Supplier: "Cennik", UnitPrice: v,
				}); err != nil {
					log.Printf("[budget] upsert %q/Cennik: %v", name, err)
				} else {
					updated++
				}
			}
			continue
		}

		// Multi-supplier mode
		for col := 1; col < len(header); col++ {
			supplier := supplierCols[col]
			if supplier == "" {
				continue
			}
			cellVal := ""
			if col < len(row) {
				cellVal = row[col]
			}
			v, ok := parsePrice(cellVal)
			if !ok {
				continue
			}
			if err := s.repo.UpsertPrice(ctx, UpsertPriceRequest{
				ItemName: name, Supplier: supplier, UnitPrice: v,
			}); err != nil {
				log.Printf("[budget] upsert %q/%s: %v", name, supplier, err)
			} else {
				updated++
			}
		}
	}
	return updated, nil
}
