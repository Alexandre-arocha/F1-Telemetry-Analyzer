import psycopg

from .models import RaceRec

_TELEMETRY_SQL = """
INSERT INTO telemetry (lap_id, step_m, time_s, speed, rpm, gear, throttle, brake, x, y)
VALUES (%s, %s, %s::real[], %s::real[], %s::real[], %s::smallint[], %s::real[],
        %s::smallint[], %s::real[], %s::real[])
"""


def save_race(database_url: str, race: RaceRec) -> None:
    """Replace the race (and everything under it) in one transaction, so re-runs are idempotent."""
    with psycopg.connect(database_url) as conn, conn.cursor() as cur:
        cur.execute("DELETE FROM races WHERE year = %s AND round = %s", (race.year, race.round))
        cur.execute(
            "INSERT INTO races (year, round, name, circuit, country, event_date, total_laps) "
            "VALUES (%s, %s, %s, %s, %s, %s, %s) RETURNING id",
            (race.year, race.round, race.name, race.circuit, race.country, race.event_date, race.total_laps),
        )
        race_id = cur.fetchone()[0]
        for d in race.drivers:
            cur.execute(
                "INSERT INTO drivers (race_id, code, number, full_name, team) "
                "VALUES (%s, %s, %s, %s, %s) RETURNING id",
                (race_id, d.code, d.number, d.full_name, d.team),
            )
            driver_id = cur.fetchone()[0]
            for lap in d.laps:
                cur.execute(
                    "INSERT INTO laps (race_id, driver_id, lap_number, lap_time_ms, s1_ms, s2_ms, s3_ms, "
                    "compound, tyre_life, stint, position, clean, has_telemetry) "
                    "VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s) RETURNING id",
                    (race_id, driver_id, lap.number, lap.lap_time_ms, lap.s1_ms, lap.s2_ms, lap.s3_ms,
                     lap.compound, lap.tyre_life, lap.stint, lap.position, lap.clean,
                     lap.telemetry is not None),
                )
                lap_id = cur.fetchone()[0]
                if lap.telemetry:
                    t = lap.telemetry
                    cur.execute(_TELEMETRY_SQL, (
                        lap_id, t["step_m"], t["time_s"], t["speed"], t["rpm"], t["gear"],
                        t["throttle"], t["brake"], t["x"], t["y"],
                    ))
