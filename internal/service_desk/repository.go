package service_desk

import (
	"fmt"
	"time"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type ServiceDeskRepository struct {
	Repository *repository.Repository
}

func NewServiceDeskRepository(r *repository.Repository) *ServiceDeskRepository {
	return &ServiceDeskRepository{Repository: r}
}

func (r *ServiceDeskRepository) CreateRequest(request *Request) error {

	row := goqu.Record{
		"title":       request.Title,
		"description": request.Description,
		"type":        request.Type,
		"status":      request.Status,
		"created_by":  request.CreatedBy,
		"priority":    request.Priority,
	}

	if request.CreatedByID != nil {
		row["created_by_id"] = request.CreatedByID
	}

	if request.AssignedTo != nil {
		row["assigned_to_id"] = request.AssignedTo
	}

	if request.Location != nil {
		row["location"] = request.Location
	}

	query := r.Repository.GoquDBWrapper.Insert("service_desk_requests").Rows(row).Returning("id")

	if _, err := query.Executor().ScanVal(&request.ID); err != nil {
		return fmt.Errorf("unable to execute SQL: %w", err)
	}

	return nil
}

func (r *ServiceDeskRepository) UpdateRequestStatus(request *Request) error {
	query := r.Repository.GoquDBWrapper.Update("service_desk_requests").
		Set(goqu.Record{
			"status":     request.Status,
			"updated_at": request.UpdatedAt,
		}).
		Where(goqu.Ex{"id": request.ID})

	if _, err := query.Executor().ScanVal(&request.ID); err != nil {
		return fmt.Errorf("unable to execute SQL: %w", err)
	}

	return nil
}

func (r *ServiceDeskRepository) UpdateRequestAssignedTo(requestID int, assignedToID int, UpdatedAt time.Time) error {
	query := r.Repository.GoquDBWrapper.Update("service_desk_requests").
		Set(goqu.Record{
			"assigned_to_id": assignedToID,
			"updated_at":     UpdatedAt,
		}).
		Where(goqu.Ex{"id": requestID})

	if _, err := query.Executor().ScanVal(&requestID); err != nil {
		return fmt.Errorf("unable to execute SQL: %w", err)
	}

	return nil
}

func (r *ServiceDeskRepository) GetRequest(id int) (*RequestResponse, error) {
	query := r.prepareRequestQuery().Where(goqu.Ex{"sdr.id": id})

	var requestFlatResponse FlatRequestResponse

	ok, err := query.Executor().ScanStruct(&requestFlatResponse)

	requestResponse := requestFlatResponse.TransformToRequestResponse()

	if err != nil {
		return nil, fmt.Errorf("unable to execute SQL: %w", err)
	}

	if !ok {
		return nil, fmt.Errorf("request not found")
	}

	return requestResponse, nil
}

func (r *ServiceDeskRepository) CreateComment(comment *RequestComment) (int, error) {
	query := r.Repository.GoquDBWrapper.Insert("service_desk_request_comments").
		Rows(goqu.Record{
			"request_id": comment.RequestID,
			"comment":    comment.Content,
			"user_id":    comment.UserID,
			"created_at": comment.CreatedAt,
		}).
		Returning("id")

	var commentID int

	if _, err := query.Executor().ScanVal(&commentID); err != nil {
		return 0, fmt.Errorf("unable to execute SQL: %w", err)
	}

	return commentID, nil
}

func (r *ServiceDeskRepository) RequestsExists(id int) (bool, error) {
	query := r.Repository.GoquDBWrapper.Select(goqu.I("id")).From("service_desk_requests").Where(goqu.Ex{"id": id})

	result, err := query.Executor().Exec()

	if err != nil {
		return false, fmt.Errorf("unable to execute SQL: %w", err)
	}

	rows, err := result.RowsAffected()

	if err != nil {
		return false, fmt.Errorf("unable to get rows affected: %w", err)
	}

	return rows > 0, nil
}

func (r *ServiceDeskRepository) GetRequests(status string, limit int, offset int) ([]*RequestResponse, error) {

	query := r.prepareRequestQuery()

	if status != "" {
		query = query.Where(goqu.Ex{"sdr.status": status})
	}
	query = query.Limit(uint(limit)).Offset(uint(offset)).Order(goqu.I("sdr.id").Asc())

	var flatRequests []FlatRequestResponse

	err := query.Executor().ScanStructs(&flatRequests)

	if err != nil {
		return nil, fmt.Errorf("unable to execute SQL: %w", err)
	}

	requests := make([]*RequestResponse, len(flatRequests))

	for i, flatRequest := range flatRequests {
		requests[i] = flatRequest.TransformToRequestResponse()
	}

	return requests, nil
}

func (r *ServiceDeskRepository) GetComment(id int) (*Comment, error) {
	query := r.Repository.GoquDBWrapper.Select(
		goqu.I("sc.id"),
		goqu.I("sc.request_id"),
		goqu.I("sc.comment"),
		goqu.I("sc.created_at"),
		goqu.I("sc.user_id"),
		goqu.I("cu.username").As("comment_user_username"),
		goqu.I("cu.fullname").As("comment_user_fullname"),
	).
		From(goqu.T("service_desk_request_comments").As("sc")).
		LeftJoin(goqu.T("users").As("cu"), goqu.On(goqu.Ex{"sc.user_id": goqu.I("cu.id")})).
		Where(goqu.Ex{"sc.id": id})

	var comment FlatComment

	ok, err := query.Executor().ScanStruct(&comment)

	if err != nil {
		return nil, fmt.Errorf("unable to execute SQL: %w", err)
	}

	if !ok {
		return nil, fmt.Errorf("comment not found")
	}

	return &Comment{
		ID:        comment.ID,
		RequestID: comment.RequestID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
		User: &User{
			ID:       comment.UserID,
			Username: comment.Username,
			Fullname: comment.Fullname,
		},
	}, nil
}

func (r *ServiceDeskRepository) GetComments(requestID int) ([]*Comment, error) {
	query := r.Repository.GoquDBWrapper.Select(
		goqu.I("sc.id"),
		goqu.I("sc.request_id"),
		goqu.I("sc.comment"),
		goqu.I("sc.created_at"),
		goqu.I("sc.user_id"),
		goqu.I("cu.username").As("comment_user_username"),
		goqu.I("cu.fullname").As("comment_user_fullname"),
	).
		From(goqu.T("service_desk_request_comments").As("sc")).
		LeftJoin(goqu.T("users").As("cu"), goqu.On(goqu.Ex{"sc.user_id": goqu.I("cu.id")})).
		Where(goqu.Ex{"sc.request_id": requestID})

	var comments []FlatComment
	err := query.Executor().ScanStructs(&comments)

	if err != nil {
		return nil, fmt.Errorf("unable to execute SQL: %w", err)
	}

	commentsResponse := make([]*Comment, len(comments))

	for i, comment := range comments {
		commentsResponse[i] = &Comment{
			ID:        comment.ID,
			RequestID: comment.RequestID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
			User: &User{
				ID:       comment.UserID,
				Username: comment.Username,
				Fullname: comment.Fullname,
			},
		}
	}

	return commentsResponse, nil
}

func (r *ServiceDeskRepository) prepareRequestQuery() *goqu.SelectDataset {
	query := r.Repository.GoquDBWrapper.Select(
		goqu.I("sdr.id"),
		goqu.I("sdr.title"),
		goqu.I("sdr.description"),
		goqu.I("sdr.status"),
		goqu.I("sdr.type"),
		goqu.I("sdr.created_by"),
		goqu.I("sdr.created_at"),
		goqu.I("sdr.updated_at"),
		goqu.I("sdr.priority"),
		goqu.I("sdr.location"),
		goqu.I("sdr.created_by_id"),
		goqu.I("cu.username").As("request_user_username"),
		goqu.I("cu.fullname").As("request_user_fullname"),
		goqu.I("sdr.assigned_to_id"),
		goqu.I("au.username").As("request_assigned_to_username"),
		goqu.I("au.fullname").As("request_assigned_to_fullname"),
	).
		From(goqu.T("service_desk_requests").As("sdr")).
		LeftJoin(goqu.T("users").As("cu"), goqu.On(goqu.Ex{"sdr.created_by_id": goqu.I("cu.id")})).
		LeftJoin(goqu.T("users").As("au"), goqu.On(goqu.Ex{"sdr.assigned_to_id": goqu.I("au.id")}))

	return query
}
