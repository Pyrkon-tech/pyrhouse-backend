-- Remove the 1h demontage slots added by the up migration (no restore of previous state)
DELETE FROM schedule_slots
WHERE slot_type = 'demontage'
  AND schedule_id IN (SELECT id FROM schedules WHERE status = 'active');
