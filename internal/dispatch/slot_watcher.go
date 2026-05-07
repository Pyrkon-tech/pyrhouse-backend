package dispatch

import (
	"database/sql"
	"log"
	"time"
)

// SlotWatcher polls the DB every minute and fires duty_roster_changed when a
// slot boundary (start or end) falls in the elapsed window.
type SlotWatcher struct {
	db          *sql.DB
	broadcaster *Broadcaster
	stop        chan struct{}
}

func NewSlotWatcher(db *sql.DB, b *Broadcaster) *SlotWatcher {
	return &SlotWatcher{
		db:          db,
		broadcaster: b,
		stop:        make(chan struct{}),
	}
}

func (w *SlotWatcher) Start() {
	go w.run()
}

func (w *SlotWatcher) Stop() {
	close(w.stop)
}

func (w *SlotWatcher) run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	last := time.Now()

	for {
		select {
		case <-w.stop:
			return
		case now := <-ticker.C:
			w.checkBoundaries(last, now)
			last = now
		}
	}
}

type slotBoundary struct {
	slotID int
	reason string
}

func (w *SlotWatcher) checkBoundaries(from, to time.Time) {
	rows, err := w.db.Query(`
		SELECT id, 'slot_started' AS reason FROM schedule_slots
		WHERE start_time > $1 AND start_time <= $2
		UNION ALL
		SELECT id, 'slot_ended' AS reason FROM schedule_slots
		WHERE end_time > $1 AND end_time <= $2
	`, from, to)
	if err != nil {
		log.Printf("[dispatch/slot-watcher] query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var b slotBoundary
		if err := rows.Scan(&b.slotID, &b.reason); err != nil {
			log.Printf("[dispatch/slot-watcher] scan error: %v", err)
			continue
		}
		w.broadcaster.BroadcastRosterChanged(b.reason, b.slotID)
		log.Printf("[dispatch/slot-watcher] %s for slot %d", b.reason, b.slotID)
	}
}
