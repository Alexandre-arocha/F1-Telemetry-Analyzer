import numpy as np
import pytest

from ingest.normalize import is_clean, lap_series, resample_lap


def synthetic_lap(n=900, length=4500.0):
    d = np.linspace(0, length, n)
    return d, d / 60.0, {
        "speed": 200 + 50 * np.sin(d / 300),
        "rpm": np.full(n, 11000.0),
        "gear": np.where(d < 2000, 6, 3),
        "throttle": np.full(n, 100.0),
        "brake": (d > 1000) & (d < 1100),
        "x": d,
        "y": -d,
    }


def test_resample_puts_lap_on_fixed_grid():
    d, t, ch = synthetic_lap()
    out = resample_lap(d, t, ch, step=5.0)
    assert len(out["speed"]) == 900  # 4500 m / 5 m
    assert out["time_s"][0] == 0.0
    assert all(b >= a for a, b in zip(out["time_s"], out["time_s"][1:]))
    assert set(out["brake"]) == {0, 1}
    assert all(isinstance(g, int) for g in out["gear"])
    assert out["brake"][int(1050 / 5)] == 1 and out["brake"][int(500 / 5)] == 0


def test_resample_survives_duplicate_and_backwards_distance():
    d, t, ch = synthetic_lap()
    d[10] = d[9]  # duplicate
    d[20] = d[19] - 1  # glitch going backwards
    out = resample_lap(d, t, ch)
    assert len(out["time_s"]) > 800


def test_resample_replaces_nan_so_json_stays_valid():
    d, t, ch = synthetic_lap()
    ch["x"][100] = np.nan
    out = resample_lap(d, t, ch)
    assert not any(np.isnan(out["x"]))


def test_resample_rejects_incomplete_laps():
    d, t, ch = synthetic_lap(length=500.0)
    with pytest.raises(ValueError):
        resample_lap(d, t, ch)
    with pytest.raises(ValueError):
        resample_lap(d[:5], t[:5], {k: v[:5] for k, v in ch.items()})


def test_lap_series_integrates_distance_from_speed():
    t0, dur = 1000.0, 90.0
    t = np.arange(t0 - 1, t0 + dur + 1, 0.27)
    car = {"t": t, "speed": np.full(t.size, 180.0), "rpm": np.full(t.size, 10000.0),
           "gear": np.full(t.size, 7.0), "throttle": np.full(t.size, 100.0),
           "brake": np.where(t < t0 + 10, 100, 0)}
    lt = np.arange(t0 - 1, t0 + dur + 1, 0.3)
    loc = {"t": lt, "x": lt - t0, "y": np.zeros(lt.size)}
    ts, dist, ch = lap_series(car, loc, t0, dur)
    assert ts[0] == 0 and ts[-1] == dur
    assert dist[-1] == pytest.approx(180 / 3.6 * dur, rel=1e-3)  # 4500 m
    assert set(np.unique(ch["brake"])) <= {0.0, 1.0} or 0 < ch["brake"].max() <= 1


def test_lap_series_rejects_samples_that_miss_the_lap():
    t = np.arange(1000.0, 1050.0, 0.3)  # stops 40 s before the lap ends
    car = {"t": t, "speed": np.full(t.size, 100.0), "rpm": t, "gear": t, "throttle": t, "brake": t}
    with pytest.raises(ValueError):
        lap_series(car, {"t": t, "x": t, "y": t}, 1000.0, 90.0)


@pytest.mark.parametrize(
    "args, expected",
    [
        ((5, 90000, False, 91000), True),
        ((1, 90000, False, 91000), False),  # first lap
        ((5, None, False, 91000), False),  # no lap time
        ((5, 90000, True, 91000), False),  # pit in/out
        ((5, 120000, False, 91000), False),  # safety-car pace, above 107%
    ],
)
def test_is_clean(args, expected):
    assert is_clean(*args) is expected
