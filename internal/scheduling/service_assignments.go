package scheduling

import (
	"strings"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type DuplicateAssignmentError struct {
	VolunteerID int
	SlotID      int
}

func (e *DuplicateAssignmentError) Error() string {
	return "already_assigned"
}

func isDuplicateError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "UNIQUE constraint")
}

func (s *Service) AddAssignment(req AddAssignmentRequest) (*AssignmentDetail, error) {
	assignment, err := s.repo.CreateAssignment(req.SlotID, req.VolunteerID)
	if err != nil {
		if isDuplicateError(err) {
			return nil, &DuplicateAssignmentError{VolunteerID: req.VolunteerID, SlotID: req.SlotID}
		}
		return nil, err
	}

	row, err := s.repo.GetAssignmentWithNickname(assignment.ID)
	if err != nil || row == nil {
		return &AssignmentDetail{
			ID:          assignment.ID,
			SlotID:      assignment.SlotID,
			VolunteerID: assignment.VolunteerID,
		}, nil
	}
	return &AssignmentDetail{
		ID:          row.ID,
		SlotID:      row.SlotID,
		VolunteerID: row.VolunteerID,
		Nickname:    row.Nickname,
	}, nil
}

func (s *Service) DeleteAssignment(assignmentID int) error {
	exists, err := s.repo.AssignmentExists(assignmentID)
	if err != nil {
		return err
	}
	if !exists {
		return nil // idempotent — already gone
	}
	return s.repo.DeleteAssignment(assignmentID)
}

func (s *Service) MoveAssignment(req MoveAssignmentRequest) (*MoveResponse, error) {
	a, err := s.repo.GetAssignmentWithNickname(req.AssignmentID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAssignmentNotFound
	}

	deletedID := a.ID

	// Delete old assignment, create new one in a transaction
	var newAssignment Assignment
	err = repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		if err := s.repo.DeleteAssignmentTx(tx, deletedID); err != nil {
			return err
		}
		created, err := s.repo.CreateAssignmentTx(tx, req.ToSlotID, a.VolunteerID)
		if err != nil {
			if isDuplicateError(err) {
				return &DuplicateAssignmentError{VolunteerID: a.VolunteerID, SlotID: req.ToSlotID}
			}
			return err
		}
		newAssignment = *created
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &MoveResponse{
		DeletedAssignmentID: deletedID,
		CreatedAssignment: AssignmentDetail{
			ID:          newAssignment.ID,
			SlotID:      newAssignment.SlotID,
			VolunteerID: newAssignment.VolunteerID,
			Nickname:    a.Nickname,
		},
	}, nil
}

func (s *Service) SwapAssignments(req SwapRequest) (*SwapResponse, error) {
	a, err := s.repo.GetAssignmentWithNickname(req.AssignmentA)
	if err != nil {
		return nil, err
	}
	b, err := s.repo.GetAssignmentWithNickname(req.AssignmentB)
	if err != nil {
		return nil, err
	}
	if a == nil || b == nil {
		return nil, ErrAssignmentsNotFound
	}

	var newA, newB Assignment
	err = repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		if err := s.repo.DeleteAssignmentTx(tx, a.ID); err != nil {
			return err
		}
		if err := s.repo.DeleteAssignmentTx(tx, b.ID); err != nil {
			return err
		}
		created, err := s.repo.CreateAssignmentTx(tx, b.SlotID, a.VolunteerID)
		if err != nil {
			return err
		}
		newA = *created
		created, err = s.repo.CreateAssignmentTx(tx, a.SlotID, b.VolunteerID)
		if err != nil {
			return err
		}
		newB = *created
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &SwapResponse{
		AssignmentA: AssignmentDetail{ID: newA.ID, SlotID: newA.SlotID, VolunteerID: newA.VolunteerID, Nickname: a.Nickname},
		AssignmentB: AssignmentDetail{ID: newB.ID, SlotID: newB.SlotID, VolunteerID: newB.VolunteerID, Nickname: b.Nickname},
	}, nil
}
