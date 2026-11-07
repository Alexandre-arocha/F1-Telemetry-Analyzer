from dataclasses import dataclass, field


@dataclass
class LapRec:
    number: int
    lap_time_ms: int | None
    s1_ms: int | None
    s2_ms: int | None
    s3_ms: int | None
    compound: str
    tyre_life: int | None
    stint: int | None
    position: int | None
    clean: bool
    telemetry: dict | None = None


@dataclass
class DriverRec:
    code: str
    number: str
    full_name: str
    team: str
    team_colour: str = ""
    headshot_url: str = ""
    laps: list[LapRec] = field(default_factory=list)


@dataclass
class RaceRec:
    year: int
    round: int
    name: str
    circuit: str
    country: str
    event_date: str | None
    total_laps: int | None
    drivers: list[DriverRec] = field(default_factory=list)
