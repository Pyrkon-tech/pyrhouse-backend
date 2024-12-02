package auditlog

import (
	"encoding/json"
	"fmt"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type AuditLogRepository struct {
	repository *repository.Repository
}

func (r *AuditLogRepository) PersistLog(auditlog models.AuditLog, auditLogData interface{}) error {
	// all what is required resourceID int, resourceType, action string, data interface{}, userID *int

	dataJSON, err := json.Marshal(auditLogData) // Convert data to JSON
	if err != nil {
		return fmt.Errorf("failed to marshal audit log data: %w", err)
	}

	query := r.repository.GoquDBWrapper.Insert("audit_logs").
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

func NewRepository(r *repository.Repository) *AuditLogRepository {
	return &AuditLogRepository{repository: r}
}
