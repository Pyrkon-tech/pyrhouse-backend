BEGIN;

CREATE TABLE schedules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    event_start DATE NOT NULL,
    event_end DATE NOT NULL,
    festival_start TIMESTAMP NOT NULL,
    festival_end TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE schedule_volunteers (
    id SERIAL PRIMARY KEY,
    schedule_id INT NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id),
    nickname VARCHAR(255) NOT NULL,
    city VARCHAR(255),
    target_hours INT NOT NULL DEFAULT 14,
    available_from TIMESTAMP NOT NULL,
    available_to TIMESTAMP NOT NULL,
    notes TEXT,
    assigned_hours DECIMAL(4,1) NOT NULL DEFAULT 0,
    UNIQUE(schedule_id, nickname)
);

CREATE TABLE schedule_slots (
    id SERIAL PRIMARY KEY,
    schedule_id INT NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    slot_type VARCHAR(20) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    credit_hours DECIMAL(4,1) NOT NULL,
    capacity INT NOT NULL DEFAULT 2,
    label VARCHAR(100)
);

CREATE TABLE schedule_assignments (
    id SERIAL PRIMARY KEY,
    slot_id INT NOT NULL REFERENCES schedule_slots(id) ON DELETE CASCADE,
    volunteer_id INT NOT NULL REFERENCES schedule_volunteers(id) ON DELETE CASCADE,
    UNIQUE(slot_id, volunteer_id)
);

CREATE INDEX idx_schedule_volunteers_schedule ON schedule_volunteers(schedule_id);
CREATE INDEX idx_schedule_slots_schedule ON schedule_slots(schedule_id);
CREATE INDEX idx_schedule_assignments_slot ON schedule_assignments(slot_id);
CREATE INDEX idx_schedule_assignments_volunteer ON schedule_assignments(volunteer_id);

COMMIT;
