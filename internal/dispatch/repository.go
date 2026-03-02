package dispatch

import (
	"database/sql"
	"fmt"
	"strings"
	"warehouse/internal/repository"
)

// Volunteer is an active user enriched with real-time dispatch status.
type Volunteer struct {
	ID              int     `json:"id"`
	Username        string  `json:"username"`
	Fullname        *string `json:"fullname"`
	DiscordUsername *string `json:"discord_username"`
	AvatarURL       *string `json:"avatar_url"`
	// Status: "available" | "on_mission"
	// Derived from transfer_users → transfers → equipment_request_quests.
	Status         string  `json:"status"`
	CurrentMission *string `json:"current_mission"`
}

// baseVolunteerQuery returns active users enriched with on_mission status derived
// from transfer_users → transfers → equipment_request_quests (status = 'in_progress').
const baseVolunteerQuery = `
	SELECT
		u.id,
		u.username,
		u.fullname,
		u.discord_username,
		u.avatar_url,
		CASE
			WHEN aq.user_id IS NOT NULL THEN 'on_mission'
			ELSE 'available'
		END AS status,
		CASE
			WHEN aq.user_id IS NOT NULL
			THEN aq.destination_pavilion
			     || COALESCE(' - ' || NULLIF(TRIM(aq.destination_location), ''), '')
			ELSE NULL
		END AS current_mission
	FROM users u
	LEFT JOIN (
		SELECT DISTINCT ON (tu.user_id)
			tu.user_id,
			q.destination_pavilion,
			q.destination_location
		FROM transfer_users tu
		JOIN transfers t                  ON tu.transfer_id = t.id
		JOIN equipment_request_quests q   ON q.transfer_id  = t.id
		WHERE q.status = 'in_progress'
		ORDER BY tu.user_id, t.id DESC
	) aq ON u.id = aq.user_id
	WHERE u.active = true
	ORDER BY u.id ASC
`

type Repository struct {
	db *sql.DB
}

func NewRepository(r *repository.Repository) *Repository {
	return &Repository{db: r.DB}
}

// GetVolunteers returns active users with derived dispatch status.
// statusFilter limits results to the given statuses (comma-separated values from caller).
// Empty slice returns all statuses.
func (r *Repository) GetVolunteers(statusFilter []string) ([]Volunteer, error) {
	var (
		query string
		args  []interface{}
	)

	if len(statusFilter) > 0 {
		placeholders := make([]string, len(statusFilter))
		args = make([]interface{}, len(statusFilter))
		for i, s := range statusFilter {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = s
		}
		query = fmt.Sprintf(
			"SELECT * FROM (%s) v WHERE v.status IN (%s)",
			baseVolunteerQuery,
			strings.Join(placeholders, ", "),
		)
	} else {
		query = baseVolunteerQuery
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch volunteers: %w", err)
	}
	defer rows.Close()

	var volunteers []Volunteer
	for rows.Next() {
		var v Volunteer
		if err := rows.Scan(
			&v.ID,
			&v.Username,
			&v.Fullname,
			&v.DiscordUsername,
			&v.AvatarURL,
			&v.Status,
			&v.CurrentMission,
		); err != nil {
			return nil, fmt.Errorf("failed to scan volunteer: %w", err)
		}
		volunteers = append(volunteers, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row error: %w", err)
	}
	if volunteers == nil {
		volunteers = []Volunteer{}
	}
	return volunteers, nil
}
