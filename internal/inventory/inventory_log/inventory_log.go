package inventorylog

import (
	"time"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/models"
)

type InventoryLog struct {
	a *auditlog.Auditlog
}

func NewInventoryLog(a *auditlog.Auditlog) *InventoryLog {
	return &InventoryLog{a: a}
}

func (s *InventoryLog) CreateDeliveryLocationAssetLog(action string, asset *models.Asset, latitude float64, longitude float64, timestamp time.Time) {
	s.a.Log(
		action,
		map[string]interface{}{
			"asset_id": asset.ID,
			"msg":      "Ostatnia znana lokalizacja",
			"location": map[string]interface{}{
				"location_id": asset.Location.ID,
				"latitude":    latitude,
				"longitude":   longitude,
				"timestamp":   timestamp,
			},
		},
		asset,
	)
}

func (s *InventoryLog) CreateAssetAuditLogEntry(action string, asset *models.Asset, msg string) {
	s.a.Log(
		action,
		map[string]interface{}{
			"asset_id": asset.ID,
			"msg":      msg,
		},
		asset,
	)
}

func (s *InventoryLog) CreateTransferAuditLogEntry(action string, ts *models.Transfer) {
	// Define log messages
	logMessages := map[string]map[string]string{
		"delivered": {
			"transferMessage": "Transfer completed",
			"assetsMessage":   "Asset moved to transfer locations",
			"stocksMessage":   "Stock Items moved to transfer locations",
		},
		"in_transfer": {
			"transferMessage": "Transfer registered",
			"assetsMessage":   "Assets in transport",
			"stocksMessage":   "Stock Items in transport",
		},
		"cancelled": {
			"transferMessage": "Transfer cancelled",
			"assetsMessage":   "Assets returned to original location",
			"stocksMessage":   "Stock Items returned to original location",
		},
	}

	messages, ok := logMessages[action]
	if !ok {
		return
	}

	s.a.Log(
		action,
		map[string]interface{}{
			"transfer_id":      ts.ID,
			"from_location_id": ts.FromLocation.ID,
			"to_location_id":   ts.ToLocation.ID,
			"msg":              messages["transferMessage"],
		},
		ts,
	)

	for _, asset := range ts.AssetsCollection {
		asset := asset
		s.a.Log(
			action,
			map[string]interface{}{
				"transfer_id":      ts.ID,
				"from_location_id": ts.FromLocation.ID,
				"to_location_id":   ts.ToLocation.ID,
				"msg":              messages["assetsMessage"],
			},
			&asset,
		)
	}

	for _, stock := range ts.StockItemsCollection {
		stock := stock
		s.a.Log(
			action,
			map[string]interface{}{
				"transfer_id":      ts.ID,
				"from_location_id": ts.FromLocation.ID,
				"to_location_id":   ts.ToLocation.ID,
				"quantity":         stock.Quantity,
				"msg":              messages["stocksMessage"],
			},
			stock,
		)
	}
}

func (s *InventoryLog) CreateTransferUserLogEntry(action string, transferID int, user *models.TransferUser) {
	s.a.Log(
		action,
		map[string]interface{}{
			"transfer_id": transferID,
			"user_id":     user.UserID,
			"msg":         "Użytkownik przypisany do questa dostawy",
		},
		user,
	)
}
