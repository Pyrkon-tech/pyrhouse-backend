package service_desk

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrRequestNotFound = errors.New("zgłoszenie nie znalezione")
	ErrInvalidStatus   = errors.New("nieprawidłowy status")
	ErrInvalidType     = errors.New("nieprawidłowy typ zgłoszenia")
)

type Service struct {
	repository *ServiceDeskRepository

	sseMu      sync.RWMutex
	sseClients map[chan ServiceDeskEvent]struct{}
}

func NewService(repository *ServiceDeskRepository) *Service {
	return &Service{
		repository: repository,
		sseClients: make(map[chan ServiceDeskEvent]struct{}),
	}
}

// Subscribe registers a channel to receive service desk events over SSE.
// The returned channel is buffered (capacity 10) to avoid blocking the caller.
func (s *Service) Subscribe() chan ServiceDeskEvent {
	ch := make(chan ServiceDeskEvent, 10)
	s.sseMu.Lock()
	s.sseClients[ch] = struct{}{}
	s.sseMu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the broadcaster and closes it.
func (s *Service) Unsubscribe(ch chan ServiceDeskEvent) {
	s.sseMu.Lock()
	delete(s.sseClients, ch)
	close(ch)
	s.sseMu.Unlock()
}

// broadcastEvent sends an event to all connected SSE clients.
// Slow clients are skipped (non-blocking send).
func (s *Service) broadcastEvent(event ServiceDeskEvent) {
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()

	for ch := range s.sseClients {
		select {
		case ch <- event:
		default: // skip slow client
		}
	}
}

func (s *Service) CreateRequest(req *Request) error {
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Status = StatusNew

	err := s.repository.CreateRequest(req)
	if err != nil {
		return err
	}

	go s.broadcastEvent(ServiceDeskEvent{
		Type:        "request_created",
		RequestID:   req.ID,
		RequestType: req.Type,
	})

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

		if err := s.repository.UpdateRequestStatus(&updateRequest); err != nil {
			return err
		}

		go s.broadcastEvent(ServiceDeskEvent{
			Type:      "request_updated",
			RequestID: requestID,
			Field:     "status",
			Value:     newStatus,
		})

		return nil
	default:
		return ErrInvalidStatus
	}
}

func (s *Service) AssignRequest(requestID int, userID int) error {

	exists, err := s.repository.RequestsExists(requestID)
	if err != nil {
		return err
	}

	if !exists {
		return ErrRequestNotFound
	}

	UpdatedAt := time.Now()

	if err := s.repository.UpdateRequestAssignedTo(requestID, userID, UpdatedAt); err != nil {
		return err
	}

	go s.broadcastEvent(ServiceDeskEvent{
		Type:      "request_updated",
		RequestID: requestID,
		Field:     "assigned_to",
	})

	return nil
}

func (s *Service) AddComment(requestID int, content string, userID int) (*Comment, error) {
	commentReq := &RequestComment{
		RequestID: requestID,
		Content:   content,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	commentID, err := s.repository.CreateComment(commentReq)
	if err != nil {
		return nil, err
	}

	comment, err := s.repository.GetComment(commentID)
	if err != nil {
		return nil, err
	}

	go s.broadcastEvent(ServiceDeskEvent{
		Type:      "comment_added",
		RequestID: requestID,
	})

	return comment, nil
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
