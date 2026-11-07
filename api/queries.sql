-- name: ListRaces :many
SELECT id, year, round, name, circuit, country, event_date, total_laps
FROM races ORDER BY year DESC, round DESC;

-- name: GetRace :one
SELECT id, year, round, name, circuit, country, event_date, total_laps
FROM races WHERE id = $1;

-- name: ListDrivers :many
SELECT id, code, number, full_name, team FROM drivers WHERE race_id = $1 ORDER BY code;

-- name: ListLaps :many
SELECT l.id, d.code AS driver_code, l.lap_number, l.lap_time_ms, l.s1_ms, l.s2_ms, l.s3_ms,
       l.compound, l.tyre_life, l.stint, l.position, l.clean, l.has_telemetry
FROM laps l JOIN drivers d ON d.id = l.driver_id
WHERE l.race_id = $1
ORDER BY d.code, l.lap_number;

-- name: GetLap :one
SELECT l.id, l.race_id, d.code AS driver_code, l.lap_number, l.lap_time_ms, l.has_telemetry
FROM laps l JOIN drivers d ON d.id = l.driver_id
WHERE l.id = $1;

-- name: GetTelemetry :one
SELECT lap_id, step_m, time_s, speed, rpm, gear, throttle, brake, x, y
FROM telemetry WHERE lap_id = $1;
