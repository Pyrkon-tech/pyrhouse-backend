package budget

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
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
