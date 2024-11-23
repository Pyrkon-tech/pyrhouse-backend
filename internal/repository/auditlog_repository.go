package repository

import (
	"encoding/json"
	"fmt"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

func (r *Repository) PersistLog(auditlog models.AuditLog, auditLogData interface{}) error {
	// all what is required resourceID int, resourceType, action string, data interface{}, userID *int

	dataJSON, err := json.Marshal(auditLogData) // Convert data to JSON
	if err != nil {
		return fmt.Errorf("failed to marshal audit log data: %w", err)
	}

	query := r.goquDBWrapper.Insert("audit_logs").
		Rows(goqu.Record{
			"resource_id":   auditlog.ResourceID,
			"resource_type": auditlog.ResourceType,
			"action":        auditlog.Action,
			"data":          dataJSON,
			// TODO "user_id":       auditlog.UserID,
		})

	_, err = query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}
