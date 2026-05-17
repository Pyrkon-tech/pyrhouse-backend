BEGIN;

-- Remove hourly montage slots created by the up migration.
-- Volunteer assignments cascade-delete automatically.
-- NOTE: this rollback is lossy — original 12h slots and their assignments are not restored.
DO $$
DECLARE
    sched_id INT;
BEGIN
    SELECT id INTO sched_id FROM schedules WHERE status = 'active' LIMIT 1;
    IF sched_id IS NULL THEN RETURN; END IF;

    DELETE FROM schedule_slots
    WHERE schedule_id = sched_id
      AND slot_type   = 'montage'
      AND credit_hours = 1.0;

    -- Recalculate assigned_hours after removing montage assignments
    UPDATE schedule_volunteers sv
    SET assigned_hours = (
        SELECT COALESCE(SUM(ss.credit_hours), 0)
        FROM schedule_assignments sa
        JOIN schedule_slots ss ON ss.id = sa.slot_id
        WHERE sa.volunteer_id = sv.id
    )
    WHERE sv.schedule_id = sched_id;

    UPDATE schedules SET version = version + 1 WHERE id = sched_id;
END;
$$;

DROP TABLE IF EXISTS schedule_day_windows;

COMMIT;
