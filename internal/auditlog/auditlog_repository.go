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

func (r *AuditLogRepository) GetResourceLog(id int, resourceType string) (*[]models.AuditLog, error) {
	query := r.repository.GoquDBWrapper.
		From(goqu.T("audit_logs").As("a")).
		Select(
			goqu.I("a.id").As("id"),
			goqu.I("a.resource_id").As("resource_id"),
			goqu.I("a.resource_type").As("resource_type"),
			goqu.I("a.action").As("action"),
			goqu.I("a.data").As("data"),
			goqu.I("a.created_at").As("created_at"),
			// User ID
		).
		Where(goqu.Ex{
			"a.resource_id":   id,
			"a.resource_type": resourceType,
		})
	rows, err := query.Executor().Query()

	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	var auditLogs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		rows.Scan( //if err -> handle
			&log.ID,
			&log.ResourceID,
			&log.ResourceType,
			&log.Action,
			&log.DataRaw,
			&log.CreatedAt,
		)
		log.LoadFromDB()
		auditLogs = append(auditLogs, log)
	}

	return &auditLogs, nil
}

func NewRepository(r *repository.Repository) *AuditLogRepository {
	return &AuditLogRepository{repository: r}
}
