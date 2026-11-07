"""Download a race from OpenF1 and turn it into RaceRec/DriverRec/LapRec."""
import bisect
import statistics
from datetime import datetime, timedelta

import numpy as np

from .models import DriverRec, LapRec, RaceRec
from .normalize import is_clean, lap_series, resample_lap
from .openf1 import OpenF1

CHUNK = timedelta(minutes=20)  # keeps every car_data/location request small


def _ts(iso: str) -> float:
    return datetime.fromisoformat(iso).timestamp()


def _iso(t: datetime) -> str:
    return t.isoformat()


def _ms(seconds):
    return None if seconds is None else int(round(seconds * 1000))


def _samples(api, endpoint, key, number, start, end, fields):
    """All samples of one driver between start and end, as sorted numpy arrays (t = epoch seconds)."""
    rows, cur = [], start
    while cur < end:
        nxt = min(cur + CHUNK, end)
        # keys carry the operator: "date>" + "=" is date>=, "date<" + "=" is date<= (boundaries overlap)
        rows += api.get(endpoint, session_key=key, driver_number=number, **{"date>": _iso(cur), "date<": _iso(nxt)})
        cur = nxt
    rows = sorted({r["date"]: r for r in rows}.values(), key=lambda r: r["date"])  # dedupe overlaps
    out = {"t": np.array([_ts(r["date"]) for r in rows])}
    for f in fields:
        out[f] = np.array([r[f] if r[f] is not None else 0 for r in rows], dtype=float)
    return out


def load_race(year: int, round_: int, cache_dir: str) -> RaceRec:
    api = OpenF1(cache_dir)
    sessions = sorted(api.get("sessions", year=year, session_name="Race"), key=lambda s: s["date_start"])
    if not 1 <= round_ <= len(sessions):
        raise SystemExit(f"{year} has {len(sessions)} races on OpenF1 (2023 onwards); got round {round_}")
    sess = sessions[round_ - 1]
    key = sess["session_key"]
    meeting = api.get("meetings", meeting_key=sess["meeting_key"])[0]
    print(f"{meeting['meeting_name']} {year} (session {key})", flush=True)

    laps = api.get("laps", session_key=key)
    stints = api.get("stints", session_key=key)
    pits = api.get("pit", session_key=key)
    positions = api.get("position", session_key=key)
    start = datetime.fromisoformat(sess["date_start"]) - timedelta(minutes=5)
    end = datetime.fromisoformat(sess["date_end"]) + timedelta(minutes=10)

    race = RaceRec(
        year=year,
        round=round_,
        name=meeting["meeting_name"],
        circuit=meeting.get("circuit_short_name") or sess["circuit_short_name"],
        country=sess["country_name"],
        event_date=sess["date_start"][:10],
        total_laps=max((l["lap_number"] for l in laps), default=None),
    )

    raw = {}  # (driver code, lap number) -> series, rescaled once the track length is known
    for d in sorted(api.get("drivers", session_key=key), key=lambda d: d["name_acronym"]):
        n = d["driver_number"]
        driver = DriverRec(
            code=d["name_acronym"], number=str(n), full_name=d["full_name"].title(), team=d["team_name"] or ""
        )
        dlaps = sorted((l for l in laps if l["driver_number"] == n), key=lambda l: l["lap_number"])
        if not dlaps:
            continue
        car = _samples(api, "car_data", key, n, start, end, ("speed", "rpm", "n_gear", "throttle", "brake"))
        car["gear"] = car.pop("n_gear")
        loc = _samples(api, "location", key, n, start, end, ("x", "y"))
        pos = sorted((_ts(p["date"]), p["position"]) for p in positions if p["driver_number"] == n)
        pos_t = [p[0] for p in pos]
        pit_laps = {p["lap_number"] for p in pits if p["driver_number"] == n}
        dstints = [s for s in stints if s["driver_number"] == n]
        timed = [l["lap_duration"] for l in dlaps if l["lap_duration"] and not l["is_pit_out_lap"] and l["lap_number"] not in pit_laps]
        median = statistics.median(timed) if timed else None

        for l in dlaps:
            num, dur = l["lap_number"], l["lap_duration"]
            pitted = bool(l["is_pit_out_lap"]) or num in pit_laps
            stint = next((s for s in dstints if s["lap_start"] <= num <= s["lap_end"]), None)
            pos_now, lap_end = None, None
            if l["date_start"] and dur:
                lap_end = _ts(l["date_start"]) + dur
                i = bisect.bisect_right(pos_t, lap_end) - 1
                pos_now = pos[i][1] if i >= 0 else None
            driver.laps.append(LapRec(
                number=num,
                lap_time_ms=_ms(dur),
                s1_ms=_ms(l["duration_sector_1"]),
                s2_ms=_ms(l["duration_sector_2"]),
                s3_ms=_ms(l["duration_sector_3"]),
                compound=(stint["compound"] or "").upper() if stint else "",
                tyre_life=stint["tyre_age_at_start"] + num - stint["lap_start"] if stint else None,
                stint=stint["stint_number"] if stint else None,
                position=pos_now,
                clean=is_clean(num, _ms(dur), pitted, _ms(median)),
            ))
            if lap_end:
                try:
                    raw[(driver.code, num)] = lap_series(car, loc, _ts(l["date_start"]), dur)
                except ValueError:
                    pass  # no (complete) samples for this lap, e.g. retirement
        race.drivers.append(driver)
        print(f"  {driver.code}: {len(dlaps)} laps", flush=True)

    if raw:
        # integrated distance drifts a little from lap to lap; scale every lap to the same track length
        track_m = float(np.median([dist[-1] for _, dist, _ in raw.values()]))
        for d in race.drivers:
            for lap in d.laps:
                series = raw.get((d.code, lap.number))
                if series:
                    ts, dist, ch = series
                    try:
                        lap.telemetry = resample_lap(dist * (track_m / dist[-1]), ts, ch)
                    except ValueError:
                        lap.telemetry = None
    return race
