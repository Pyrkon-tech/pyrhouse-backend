BEGIN;

-- New table: per-day custom time windows for montage/demontage slot generation.
-- When absent for a given date, the generator falls back to 08:00-20:00.
CREATE TABLE schedule_day_windows (
    id           SERIAL PRIMARY KEY,
    schedule_id  INT  NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    date         DATE NOT NULL,
    window_start TIME NOT NULL DEFAULT '08:00',
    window_end   TIME NOT NULL DEFAULT '20:00',
    UNIQUE(schedule_id, date)
);

CREATE INDEX idx_day_windows_schedule ON schedule_day_windows(schedule_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Data migration: convert existing single-slot montage days to 1h blocks.
-- Runs only when the active schedule still has old-format slots (credit_hours=7).
-- All timestamps stored in UTC (Warsaw CEST = UTC+2).
-- ─────────────────────────────────────────────────────────────────────────────
DO $$
DECLARE
    sched_id INT;
BEGIN
    SELECT id INTO sched_id FROM schedules WHERE status = 'active' LIMIT 1;
    IF sched_id IS NULL THEN RETURN; END IF;

    IF NOT EXISTS (
        SELECT 1 FROM schedule_slots
        WHERE schedule_id = sched_id AND slot_type = 'montage' AND credit_hours = 7
    ) THEN RETURN; END IF;

    -- ── 1. Register the three custom day windows ───────────────────────────
    -- Wtorek 16.06:  10:00–18:00 Warsaw
    -- Środa  17.06:  10:00–20:00 Warsaw
    -- Czwartek 18.06: 08:00–22:00 Warsaw
    INSERT INTO schedule_day_windows (schedule_id, date, window_start, window_end) VALUES
        (sched_id, '2026-06-16', '10:00', '18:00'),
        (sched_id, '2026-06-17', '10:00', '20:00'),
        (sched_id, '2026-06-18', '08:00', '22:00')
    ON CONFLICT (schedule_id, date) DO NOTHING;

    -- ── 2. Snapshot which volunteer is on which day ────────────────────────
    -- rn is 0-indexed within each old slot (sorted by nickname → deterministic pairs).
    DROP TABLE IF EXISTS _montage_vols;
    CREATE TEMP TABLE _montage_vols AS
    SELECT
        a.volunteer_id,
        (s.start_time + INTERVAL '2 hours')::date AS slot_date,
        (ROW_NUMBER() OVER (PARTITION BY s.id ORDER BY v.nickname) - 1)::int AS rn
    FROM schedule_assignments a
    JOIN schedule_slots       s ON s.id = a.slot_id
    JOIN schedule_volunteers  v ON v.id = a.volunteer_id
    WHERE s.schedule_id = sched_id
      AND s.slot_type   = 'montage'
      AND s.credit_hours = 7;

    -- ── 3. Remove old 12h montage slots (assignments cascade) ─────────────
    DELETE FROM schedule_slots
    WHERE schedule_id = sched_id
      AND slot_type   = 'montage'
      AND credit_hours = 7;

    -- ── 4. Insert hourly slots for each day window ─────────────────────────
    -- Labels show Warsaw time (UTC + 2h).

    -- Wtorek 16.06: 10:00–18:00 Warsaw = 08:00–16:00 UTC  →  8 slots
    INSERT INTO schedule_slots (schedule_id, slot_type, start_time, end_time, credit_hours, capacity, label)
    SELECT sched_id, 'montage', gs, gs + INTERVAL '1 hour', 1.0, 2,
           'Montaż - Wt ' || to_char(gs + INTERVAL '2 hours', 'HH24:MI')
               || '-' || to_char(gs + INTERVAL '3 hours', 'HH24:MI')
    FROM generate_series(
        '2026-06-16 08:00:00'::timestamp,
        '2026-06-16 15:00:00'::timestamp,
        INTERVAL '1 hour'
    ) gs;

    -- Środa 17.06: 10:00–20:00 Warsaw = 08:00–18:00 UTC  →  10 slots
    INSERT INTO schedule_slots (schedule_id, slot_type, start_time, end_time, credit_hours, capacity, label)
    SELECT sched_id, 'montage', gs, gs + INTERVAL '1 hour', 1.0, 2,
           'Montaż - Śr ' || to_char(gs + INTERVAL '2 hours', 'HH24:MI')
               || '-' || to_char(gs + INTERVAL '3 hours', 'HH24:MI')
    FROM generate_series(
        '2026-06-17 08:00:00'::timestamp,
        '2026-06-17 17:00:00'::timestamp,
        INTERVAL '1 hour'
    ) gs;

    -- Czwartek 18.06: 08:00–22:00 Warsaw = 06:00–20:00 UTC  →  14 slots
    INSERT INTO schedule_slots (schedule_id, slot_type, start_time, end_time, credit_hours, capacity, label)
    SELECT sched_id, 'montage', gs, gs + INTERVAL '1 hour', 1.0, 2,
           'Montaż - Czw ' || to_char(gs + INTERVAL '2 hours', 'HH24:MI')
               || '-' || to_char(gs + INTERVAL '3 hours', 'HH24:MI')
    FROM generate_series(
        '2026-06-18 06:00:00'::timestamp,
        '2026-06-18 19:00:00'::timestamp,
        INTERVAL '1 hour'
    ) gs;

    -- ── 5. Auto-assign volunteers to 6h blocks ─────────────────────────────
    -- Strategy: sort volunteers by nickname within each day, form pairs (cap=2).
    -- Each pair gets a 6h block; blocks are evenly spaced across the window.
    --
    --   pair_index  = rn / 2  (integer division)
    --   block_start = window_utc_start + pair_index * step_h
    --   block_end   = block_start + 6h
    --
    -- Wtorek  (8h / 3 vols, step=2h): pair0→08:00, pair1→10:00 UTC
    --   Freya(0)+Kodii(1) → 08:00–14:00 UTC = 10:00–16:00 Warsaw
    --   YaRo(2)           → 10:00–16:00 UTC = 12:00–18:00 Warsaw
    --
    -- Środa  (10h / 6 vols, step=2h): pair0→08, pair1→10, pair2→12 UTC
    --   Baja+Jomek   → 10:00–16:00 Warsaw
    --   Latul+Puszek → 12:00–18:00 Warsaw
    --   Selena+Techmox → 14:00–20:00 Warsaw
    --
    -- Czwartek (14h / 6 vols, step=4h): pair0→06, pair1→10, pair2→14 UTC
    --   Hesia+Miki      → 08:00–14:00 Warsaw
    --   Pelafina+Ritsu  → 12:00–18:00 Warsaw
    --   Theron+YaRo     → 16:00–22:00 Warsaw
    INSERT INTO schedule_assignments (volunteer_id, slot_id)
    SELECT DISTINCT mv.volunteer_id, ss.id
    FROM _montage_vols mv
    JOIN schedule_slots ss ON
        ss.schedule_id = sched_id
        AND ss.slot_type = 'montage'
        AND (ss.start_time + INTERVAL '2 hours')::date = mv.slot_date
        AND ss.start_time >= CASE mv.slot_date
            WHEN '2026-06-16'::date THEN
                '2026-06-16 08:00:00'::timestamp + make_interval(hours => mv.rn/2 * 2)
            WHEN '2026-06-17'::date THEN
                '2026-06-17 08:00:00'::timestamp + make_interval(hours => mv.rn/2 * 2)
            WHEN '2026-06-18'::date THEN
                '2026-06-18 06:00:00'::timestamp + make_interval(hours => mv.rn/2 * 4)
        END
        AND ss.start_time < CASE mv.slot_date
            WHEN '2026-06-16'::date THEN
                '2026-06-16 08:00:00'::timestamp + make_interval(hours => mv.rn/2 * 2 + 6)
            WHEN '2026-06-17'::date THEN
                '2026-06-17 08:00:00'::timestamp + make_interval(hours => mv.rn/2 * 2 + 6)
            WHEN '2026-06-18'::date THEN
                '2026-06-18 06:00:00'::timestamp + make_interval(hours => mv.rn/2 * 4 + 6)
        END;

    -- ── 6. Recalculate assigned_hours for all volunteers ───────────────────
    UPDATE schedule_volunteers sv
    SET assigned_hours = (
        SELECT COALESCE(SUM(ss.credit_hours), 0)
        FROM schedule_assignments sa
        JOIN schedule_slots ss ON ss.id = sa.slot_id
        WHERE sa.volunteer_id = sv.id
    )
    WHERE sv.schedule_id = sched_id;

    -- ── 7. Bump version to invalidate any cached client state ──────────────
    UPDATE schedules SET version = version + 1 WHERE id = sched_id;

    DROP TABLE IF EXISTS _montage_vols;
END;
$$;

COMMIT;
