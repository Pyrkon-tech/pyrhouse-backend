package service_desk

import (
	"errors"
	"log"
	"time"
)

var (
	ErrRequestNotFound = errors.New("zgłoszenie nie znalezione")
	ErrInvalidStatus   = errors.New("nieprawidłowy status")
	ErrInvalidType     = errors.New("nieprawidłowy typ zgłoszenia")
)

type Service struct {
	repository *ServiceDeskRepository
}

func NewService(repository *ServiceDeskRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateRequest(req *Request) error {
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Status = StatusNew

	err := s.repository.CreateRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ChangeStatus(requestID int, newStatus string) error {
	req, err := s.repository.GetRequest(requestID)
	if err != nil {
		return err
	}

	if req.Status == newStatus {
		return nil
	}

	switch newStatus {
	case StatusNew, StatusInProgress, StatusWaiting, StatusResolved, StatusClosed:
		var updateRequest Request
		updateRequest.ID = requestID
		updateRequest.Status = newStatus
		updateRequest.UpdatedAt = time.Now()

		return s.repository.UpdateRequestStatus(&updateRequest)
	default:
		return ErrInvalidStatus
	}
}

func (s *Service) AssignRequest(requestID int, userID int) error {
	return errors.New("not implemented")
	// req, err := s.repository.GetRequest(requestID)
	// if err != nil {
	// 	return err
	// }

	// req.AssignedTo = &userID
	// req.UpdatedAt = time.Now()
	// return s.UpdateRequest(req)
}

func (s *Service) AddComment(requestID string, content string, userID int) error {
	comment := &RequestComment{
		RequestID: requestID,
		Content:   content,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	log.Println(comment)

	// TODO: Implementacja zapisywania komentarza
	return nil
}

func (s *Service) GetRequestTypes() []RequestType {
	return []RequestType{
		{
			Type:        RequestTypeHardwareIssue,
			Name:        "Awaria sprzętu",
			Description: "Zgłoszenie problemu z działaniem sprzętu",
		},
		{
			Type:        RequestTypeReplacement,
			Name:        "Wymiana sprzętu",
			Description: "Prośba o wymianę sprzętu",
		},
		{
			Type:        RequestTypeTechnicalProblem,
			Name:        "Problem techniczny",
			Description: "Inny problem techniczny",
		},
		{
			Type:        RequestTypeOther,
			Name:        "Inne",
			Description: "Inne zgłoszenie",
		},
	}
}
