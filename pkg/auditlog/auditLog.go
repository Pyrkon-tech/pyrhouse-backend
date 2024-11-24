package auditlog

import (
	"log"

	"warehouse/internal/repository"
	"warehouse/pkg/models"
)

type Auditlog struct {
	r *repository.Repository
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

func NewAuditLog(repository *repository.Repository) *Auditlog {
	a := Auditlog{r: repository}

	return &a
}
