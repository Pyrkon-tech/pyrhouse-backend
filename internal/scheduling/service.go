package scheduling

import (
	"time"
	"warehouse/internal/integrations/googlesheets"
	"warehouse/internal/settings"
)

type Service struct {
	repo          *Repository
	sheetsHandler *googlesheets.GoogleSheetsHandler // may be nil
	settingsRepo  *settings.Repository
}

func NewService(repo *Repository, sheetsHandler *googlesheets.GoogleSheetsHandler, settingsRepo *settings.Repository) *Service {
	return &Service{repo: repo, sheetsHandler: sheetsHandler, settingsRepo: settingsRepo}
}

func calculateCreditHours(slotType string, start, end time.Time) float64 {
	if slotType == SlotTypeMontage || slotType == SlotTypeDemontage {
		return 7
	}
	return end.Sub(start).Hours()
}

// getActive returns the active schedule or an error if none exists.
func (s *Service) getActive() (*Schedule, error) {
	schedule, err := s.repo.GetActiveSchedule()
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, ErrNoActiveSchedule
	}
	return schedule, nil
}
