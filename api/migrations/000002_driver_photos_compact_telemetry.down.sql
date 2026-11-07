ALTER TABLE telemetry
    ALTER COLUMN rpm      TYPE real[] USING rpm::real[],
    ALTER COLUMN throttle TYPE real[] USING throttle::real[],
    ALTER COLUMN x        TYPE real[] USING x::real[],
    ALTER COLUMN y        TYPE real[] USING y::real[];

ALTER TABLE drivers DROP COLUMN IF EXISTS team_colour;
ALTER TABLE drivers DROP COLUMN IF EXISTS headshot_url;
