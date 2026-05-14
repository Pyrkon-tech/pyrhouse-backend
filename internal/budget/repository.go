package budget

import (
	"context"
	"sort"

	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	repo *repository.Repository
}

func NewRepository(repo *repository.Repository) *Repository {
	return &Repository{repo: repo}
}

func (r *Repository) ListPrices(ctx context.Context) ([]PriceListItem, error) {
	query, _, err := goqu.From("equipment_request_price_list").
		Order(goqu.C("item_name").Asc(), goqu.C("supplier").Asc()).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := r.repo.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PriceListItem
	for rows.Next() {
		var p PriceListItem
		if err := rows.Scan(&p.ID, &p.ItemName, &p.Supplier, &p.UnitPrice, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if items == nil {
		items = []PriceListItem{}
	}
	return items, nil
}

func (r *Repository) UpsertPrice(ctx context.Context, req UpsertPriceRequest) error {
	sql := `
		INSERT INTO equipment_request_price_list (item_name, supplier, unit_price, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (item_name, supplier) DO UPDATE
		SET unit_price  = EXCLUDED.unit_price,
		    updated_at  = NOW()`
	_, err := r.repo.DB.ExecContext(ctx, sql, req.ItemName, req.Supplier, req.UnitPrice)
	return err
}

func (r *Repository) DeletePrice(ctx context.Context, itemName, supplier string) error {
	query, _, err := goqu.Delete("equipment_request_price_list").
		Where(
			goqu.C("item_name").Eq(itemName),
			goqu.C("supplier").Eq(supplier),
		).
		ToSQL()
	if err != nil {
		return err
	}
	_, err = r.repo.DB.ExecContext(ctx, query)
	return err
}

// rawBudgetRow is a single DB result row before aggregation into BudgetItem.
type rawBudgetRow struct {
	ItemName  string
	Quantity  int
	Supplier  *string
	UnitPrice *float64
}

// GetBudgetSummary aggregates quest items by name, left-joins all supplier prices,
// and returns a fully dynamic BudgetSummary (no hardcoded supplier names).
func (r *Repository) GetBudgetSummary(ctx context.Context, filter Filter) (*BudgetSummary, error) {
	sql := `
		SELECT
			i.item_name,
			SUM(i.quantity)::int AS quantity,
			p.supplier,
			p.unit_price
		FROM equipment_request_items i
		JOIN equipment_request_quests q ON q.id = i.quest_id
		LEFT JOIN equipment_request_price_list p
			ON lower(trim(p.item_name)) = lower(trim(i.item_name))
		WHERE q.status != 'cancelled'
		  AND ($1 = '' OR lower(trim(q.budget_owner)) = lower(trim($1)))
		GROUP BY i.item_name, p.supplier, p.unit_price
		ORDER BY i.item_name, p.supplier`

	rows, err := r.repo.DB.QueryContext(ctx, sql, filter.BudgetOwner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect raw rows then pivot into BudgetItems.
	var raw []rawBudgetRow
	for rows.Next() {
		var row rawBudgetRow
		if err := rows.Scan(&row.ItemName, &row.Quantity, &row.Supplier, &row.UnitPrice); err != nil {
			return nil, err
		}
		raw = append(raw, row)
	}

	return buildSummary(raw), nil
}

// buildSummary groups raw rows (item × supplier) into BudgetSummary.
func buildSummary(raw []rawBudgetRow) *BudgetSummary {
	// Map: itemName → BudgetItem (accumulate suppliers)
	itemMap := make(map[string]*BudgetItem)
	// Map: itemName → quantity (same value repeated per supplier row)
	qtyMap := make(map[string]int)
	// Map: supplier → grand total
	supplierTotals := make(map[string]float64)

	for _, row := range raw {
		if _, ok := itemMap[row.ItemName]; !ok {
			itemMap[row.ItemName] = &BudgetItem{
				ItemName: row.ItemName,
				Prices:   []SupplierPrice{},
			}
			qtyMap[row.ItemName] = row.Quantity
		}

		if row.Supplier != nil && row.UnitPrice != nil {
			total := *row.UnitPrice * float64(row.Quantity)
			itemMap[row.ItemName].Prices = append(itemMap[row.ItemName].Prices, SupplierPrice{
				Supplier:  *row.Supplier,
				UnitPrice: *row.UnitPrice,
				Total:     total,
			})
			supplierTotals[*row.Supplier] += total
		}
	}

	// Build ordered item list
	itemNames := make([]string, 0, len(itemMap))
	for name := range itemMap {
		itemNames = append(itemNames, name)
	}
	sort.Strings(itemNames)

	summary := &BudgetSummary{
		Items: make([]BudgetItem, 0, len(itemMap)),
	}

	for _, name := range itemNames {
		item := itemMap[name]
		item.Quantity = qtyMap[name]
		summary.TotalQuantity += item.Quantity
		summary.TotalPositions++
		if len(item.Prices) == 0 {
			summary.UnpricedCount++
		}
		summary.Items = append(summary.Items, *item)
	}

	// Build ordered supplier totals
	supplierNames := make([]string, 0, len(supplierTotals))
	for s := range supplierTotals {
		supplierNames = append(supplierNames, s)
	}
	sort.Strings(supplierNames)
	for _, s := range supplierNames {
		summary.SupplierTotals = append(summary.SupplierTotals, SupplierTotal{
			Supplier: s,
			Total:    supplierTotals[s],
		})
	}
	if summary.SupplierTotals == nil {
		summary.SupplierTotals = []SupplierTotal{}
	}

	return summary
}

func (r *Repository) GetBudgetPersons(ctx context.Context) ([]string, error) {
	sql := `
		SELECT DISTINCT trim(budget_owner)
		FROM equipment_request_quests
		WHERE budget_owner IS NOT NULL AND trim(budget_owner) != ''
		ORDER BY 1`

	rows, err := r.repo.DB.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var persons []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		persons = append(persons, p)
	}
	if persons == nil {
		persons = []string{}
	}
	return persons, nil
}

// ListSuppliers returns the distinct supplier names in the price list.
func (r *Repository) ListSuppliers(ctx context.Context) ([]string, error) {
	sql := `SELECT DISTINCT supplier FROM equipment_request_price_list ORDER BY supplier`
	rows, err := r.repo.DB.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}
	if suppliers == nil {
		suppliers = []string{}
	}
	return suppliers, nil
}
