package auditlog

import (
	"log"
	"warehouse/internal/models"
)

type Persister interface {
	PersistLog(auditLog models.AuditLog, data interface{}) error
}

type Auditlog struct {
	r       Persister
	logRepo *AuditLogRepository
}

type Auditable interface {
	CreateLogView() models.AuditLog
}

func (a *Auditlog) Log(action string, data interface{}, item Auditable) {
	// TODO Handle authorized user (context?)
	auditLog := item.CreateLogView()
	auditLog.Action = action

	err := a.r.PersistLog(auditLog, data)

	if err != nil {
		log.Println("Unable to create AuditLog entry for id ", auditLog.ResourceID)
		return
	}

	log.Println("Created AuditLog entry for id ", auditLog.ResourceID)
}

func (a *Auditlog) GetLogs(id int, resourceType string) (*[]models.AuditLog, error) {
	return a.logRepo.GetResourceLog(id, resourceType)
}

func NewAuditLog(repository *AuditLogRepository) *Auditlog {
	return &Auditlog{r: repository, logRepo: repository}
}
