ALTER TABLE drivers ADD COLUMN IF NOT EXISTS headshot_url text NOT NULL DEFAULT '';
ALTER TABLE drivers ADD COLUMN IF NOT EXISTS team_colour  text NOT NULL DEFAULT '';

-- Whole numbers where the extra precision is useless (rpm, throttle %, track x/y in 1/10 m).
-- With the 10 m grid this keeps a full season well inside the free database size.
ALTER TABLE telemetry
    ALTER COLUMN rpm      TYPE smallint[] USING rpm::smallint[],
    ALTER COLUMN throttle TYPE smallint[] USING throttle::smallint[],
    ALTER COLUMN x        TYPE smallint[] USING x::smallint[],
    ALTER COLUMN y        TYPE smallint[] USING y::smallint[];
