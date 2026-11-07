CREATE TABLE races (
    id          serial PRIMARY KEY,
    year        smallint NOT NULL,
    round       smallint NOT NULL,
    name        text     NOT NULL,
    circuit     text     NOT NULL,
    country     text     NOT NULL DEFAULT '',
    event_date  date,
    total_laps  smallint,
    UNIQUE (year, round)
);

CREATE TABLE drivers (
    id        serial PRIMARY KEY,
    race_id   int  NOT NULL REFERENCES races (id) ON DELETE CASCADE,
    code      text NOT NULL,
    number    text NOT NULL DEFAULT '',
    full_name text NOT NULL DEFAULT '',
    team      text NOT NULL DEFAULT '',
    UNIQUE (race_id, code)
);

CREATE TABLE laps (
    id            serial PRIMARY KEY,
    race_id       int      NOT NULL REFERENCES races (id) ON DELETE CASCADE,
    driver_id     int      NOT NULL REFERENCES drivers (id) ON DELETE CASCADE,
    lap_number    smallint NOT NULL,
    lap_time_ms   int,
    s1_ms         int,
    s2_ms         int,
    s3_ms         int,
    compound      text     NOT NULL DEFAULT '',
    tyre_life     smallint,
    stint         smallint,
    position      smallint,
    clean         boolean  NOT NULL DEFAULT false,
    has_telemetry boolean  NOT NULL DEFAULT false,
    UNIQUE (driver_id, lap_number)
);
CREATE INDEX laps_race_idx ON laps (race_id);

-- One row per lap, resampled on a fixed distance grid: sample i is at i * step_m metres.
-- time_s is seconds since the start of the lap; brake is 0/1.
CREATE TABLE telemetry (
    lap_id   int  PRIMARY KEY REFERENCES laps (id) ON DELETE CASCADE,
    step_m   real NOT NULL,
    time_s   real[]     NOT NULL,
    speed    real[]     NOT NULL,
    rpm      real[]     NOT NULL,
    gear     smallint[] NOT NULL,
    throttle real[]     NOT NULL,
    brake    smallint[] NOT NULL,
    x        real[]     NOT NULL,
    y        real[]     NOT NULL
);
