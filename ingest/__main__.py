import argparse
import os

from .db import save_race
from .load import load_race


def main(argv=None):
    p = argparse.ArgumentParser(prog="ingest", description="Ingest one F1 race (OpenF1) into Postgres.")
    p.add_argument("--year", type=int, required=True)
    p.add_argument("--round", type=int, required=True, help="round number in the season")
    p.add_argument("--database-url", default=os.environ.get("DATABASE_URL"))
    p.add_argument("--cache-dir", default=os.environ.get("INGEST_CACHE", "cache/openf1"))
    args = p.parse_args(argv)
    if not args.database_url:
        p.error("set DATABASE_URL or pass --database-url")

    race = load_race(args.year, args.round, args.cache_dir)
    save_race(args.database_url, race)
    laps = sum(len(d.laps) for d in race.drivers)
    with_tel = sum(1 for d in race.drivers for lap in d.laps if lap.telemetry)
    print(f"saved {race.name} {race.year}: {len(race.drivers)} drivers, {laps} laps, {with_tel} with telemetry")


if __name__ == "__main__":
    main()
