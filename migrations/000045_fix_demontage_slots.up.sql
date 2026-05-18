-- Replace demontage slots for active schedule with 1h blocks 10:00–16:00
DELETE FROM schedule_slots
WHERE slot_type = 'demontage'
  AND schedule_id IN (SELECT id FROM schedules WHERE status = 'active');

INSERT INTO schedule_slots (schedule_id, slot_type, start_time, end_time, credit_hours, capacity, label)
SELECT
    s.id,
    'demontage',
    (DATE_TRUNC('day', s.event_end AT TIME ZONE 'Europe/Warsaw') AT TIME ZONE 'Europe/Warsaw')
        + (h || ' hours')::interval,
    (DATE_TRUNC('day', s.event_end AT TIME ZONE 'Europe/Warsaw') AT TIME ZONE 'Europe/Warsaw')
        + ((h + 1) || ' hours')::interval,
    1,
    2,
    'Demontaż - ' ||
        TO_CHAR(h, 'FM00') || ':00-' ||
        TO_CHAR(h + 1, 'FM00') || ':00'
FROM schedules s
CROSS JOIN generate_series(10, 15) AS h
WHERE s.status = 'active';
