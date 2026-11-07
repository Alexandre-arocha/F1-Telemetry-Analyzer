"""Pure helpers: build one lap from raw samples, put it on a fixed distance grid, flag clean laps."""
import numpy as np

STEP_M = 10.0  # the source samples at ~3.7 Hz (15-20 m apart), so a finer grid adds no information
MIN_LAP_M = 1000.0
MIN_SAMPLES = 20
CLEAN_LIMIT = 1.07  # a clean lap is within 107% of the driver's median lap


def lap_series(car, loc, t0, duration):
    """Cut one lap out of raw car/location samples (epoch seconds) starting at t0.

    car: t, speed (km/h), rpm, gear, throttle, brake (0/100); loc: t, x, y.
    Returns (time since lap start, distance in metres integrated from speed, channels).
    Distance is integrated because the API has no odometer; the caller rescales it per lap.
    """
    t = np.asarray(car["t"], dtype=float) - t0
    lt = np.asarray(loc["t"], dtype=float) - t0
    if t.size < 2 or lt.size < 2 or t[0] > 0.5 or t[-1] < duration - 0.5:
        raise ValueError("samples do not cover the lap")

    ts = np.concatenate(([0.0], t[(t > 0) & (t < duration)], [duration]))
    ch = {k: np.interp(ts, t, np.asarray(car[k], dtype=float)) for k in ("speed", "rpm", "gear", "throttle")}
    ch["brake"] = np.interp(ts, t, (np.asarray(car["brake"], dtype=float) > 0).astype(float))
    ch["x"] = np.interp(ts, lt, np.asarray(loc["x"], dtype=float))
    ch["y"] = np.interp(ts, lt, np.asarray(loc["y"], dtype=float))

    v = ch["speed"] / 3.6
    dist = np.concatenate(([0.0], np.cumsum((v[1:] + v[:-1]) / 2 * np.diff(ts))))
    return ts, dist, ch


def resample_lap(distance, time_s, channels, step=STEP_M):
    """Interpolate a lap onto distance = 0, step, 2*step, ...

    `channels` needs speed, rpm, gear, throttle, brake (0/1), x, y (same length as `distance`).
    Returns plain lists ready for Postgres arrays; time_s starts at 0.
    Raises ValueError for laps too incomplete to compare.
    """
    d = np.asarray(distance, dtype=float)
    t = np.asarray(time_s, dtype=float)
    ok = np.isfinite(d) & np.isfinite(t)
    if ok.sum() < MIN_SAMPLES:
        raise ValueError("too few telemetry samples")

    d = np.maximum.accumulate(d[ok])  # distance must never go backwards
    d, first = np.unique(d, return_index=True)  # interp needs strictly increasing x
    t = t[ok][first]
    if d[-1] - d[0] < MIN_LAP_M:
        raise ValueError("lap shorter than 1 km")
    d = d - d[0]
    grid = np.arange(0.0, d[-1], step)

    def at(values):
        v = np.asarray(values, dtype=float)[ok][first]
        return np.nan_to_num(np.interp(grid, d, v))

    return {
        "step_m": float(step),
        "time_s": np.round(np.interp(grid, d, t) - t[0], 3).tolist(),
        "speed": np.round(at(channels["speed"]), 1).tolist(),
        "rpm": np.rint(at(channels["rpm"])).astype(int).tolist(),
        "gear": np.rint(at(channels["gear"])).astype(int).tolist(),
        "throttle": np.rint(at(channels["throttle"])).astype(int).tolist(),
        "brake": (at(channels["brake"]) > 0.5).astype(int).tolist(),
        "x": np.rint(at(channels["x"])).astype(int).tolist(),
        "y": np.rint(at(channels["y"])).astype(int).tolist(),
    }


def is_clean(lap_number, lap_time_ms, pitted, median_ms):
    """Representative racing lap: timed, not the first lap, no pit, not slowed (safety car, traffic)."""
    return (
        lap_number > 1
        and lap_time_ms is not None
        and not pitted
        and median_ms is not None
        and lap_time_ms <= CLEAN_LIMIT * median_ms
    )
