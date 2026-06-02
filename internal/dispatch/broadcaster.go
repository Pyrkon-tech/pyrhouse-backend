package dispatch

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
)

// DispatchEvent is broadcast over SSE on /dispatch/stream.
// Discriminate on Type:
//
//	"volunteer_status_changed" — user changed dispatch status (UserID, Status, CurrentMission populated)
//	"duty_roster_changed"      — a slot boundary was crossed (Reason, SlotID populated)
type DispatchEvent struct {
	Type           string  `json:"type"`
	UserID         *int    `json:"user_id,omitempty"`
	Status         *string `json:"status,omitempty"`
	CurrentMission *string `json:"current_mission,omitempty"`
	Reason         *string `json:"reason,omitempty"`
	SlotID         *int    `json:"slot_id,omitempty"`
}

// Broadcaster manages SSE clients and dispatches events to them.
type Broadcaster struct {
	db      *sql.DB
	mu      sync.RWMutex
	clients map[chan DispatchEvent]struct{}
}

func NewBroadcaster(db *sql.DB) *Broadcaster {
	return &Broadcaster{
		db:      db,
		clients: make(map[chan DispatchEvent]struct{}),
	}
}

func (b *Broadcaster) Subscribe() chan DispatchEvent {
	ch := make(chan DispatchEvent, 10)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broadcaster) Unsubscribe(ch chan DispatchEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *Broadcaster) broadcast(event DispatchEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

// BroadcastRosterChanged fires duty_roster_changed when a slot boundary is crossed.
func (b *Broadcaster) BroadcastRosterChanged(reason string, slotID int) {
	b.broadcast(DispatchEvent{
		Type:   "duty_roster_changed",
		Reason: &reason,
		SlotID: &slotID,
	})
}

// BroadcastTransferDispatched fires volunteer_status_changed (on_mission) for all
// users in the given transfer. Called when a quest is dispatched (transfer created).
func (b *Broadcaster) BroadcastTransferDispatched(transferID int) {
	userIDs, mission, err := b.lookupTransfer(transferID)
	if err != nil {
		log.Printf("[dispatch/broadcaster] transfer %d lookup error: %v", transferID, err)
		return
	}
	status := "on_mission"
	for _, uid := range userIDs {
		uid := uid
		b.broadcast(DispatchEvent{
			Type:           "volunteer_status_changed",
			UserID:         &uid,
			Status:         &status,
			CurrentMission: mission,
		})
	}
}

// BroadcastTransferEnded fires volunteer_status_changed (available) for all users
// in the given transfer. Called when a transfer is completed or cancelled.
func (b *Broadcaster) BroadcastTransferEnded(transferID int) {
	userIDs, _, err := b.lookupTransfer(transferID)
	if err != nil {
		log.Printf("[dispatch/broadcaster] transfer %d lookup error: %v", transferID, err)
		return
	}
	status := "available"
	for _, uid := range userIDs {
		uid := uid
		b.broadcast(DispatchEvent{
			Type:   "volunteer_status_changed",
			UserID: &uid,
			Status: &status,
		})
	}
}

// lookupTransfer returns user IDs and mission string for a transfer.
func (b *Broadcaster) lookupTransfer(transferID int) ([]int, *string, error) {
	rows, err := b.db.Query(
		`SELECT user_id FROM transfer_users WHERE transfer_id = $1`,
		transferID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query transfer_users: %w", err)
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			return nil, nil, fmt.Errorf("scan user_id: %w", err)
		}
		userIDs = append(userIDs, uid)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var mission *string
	row := b.db.QueryRow(`
		SELECT q.destination_pavilion
		       || COALESCE(' - ' || NULLIF(TRIM(q.destination_location), ''), '')
		FROM quest_transfers qt
		JOIN equipment_request_quests q ON qt.quest_id = q.quest_id
		WHERE qt.transfer_id = $1
		LIMIT 1
	`, transferID)
	var m string
	if err := row.Scan(&m); err == nil && m != "" {
		mission = &m
	}

	return userIDs, mission, nil
}
