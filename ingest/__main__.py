import argparse
import os

from .db import save_race
from .load import count_races, load_race


def ingest(year: int, round_: int, database_url: str, cache_dir: str) -> None:
    race = load_race(year, round_, cache_dir)
    save_race(database_url, race)
    laps = sum(len(d.laps) for d in race.drivers)
    with_tel = sum(1 for d in race.drivers for lap in d.laps if lap.telemetry)
    print(f"saved {race.name} {race.year}: {len(race.drivers)} drivers, {laps} laps, {with_tel} with telemetry")


def main(argv=None):
    p = argparse.ArgumentParser(prog="ingest", description="Ingest F1 races (OpenF1) into Postgres.")
    p.add_argument("--year", type=int, required=True)
    p.add_argument("--round", type=int, help="round number in the season")
    p.add_argument("--all", action="store_true", help="every race of the season, one after another")
    p.add_argument("--database-url", default=os.environ.get("DATABASE_URL"))
    p.add_argument("--cache-dir", default=os.environ.get("INGEST_CACHE", "cache/openf1"))
    args = p.parse_args(argv)
    if not args.database_url:
        p.error("set DATABASE_URL or pass --database-url")
    if args.all == (args.round is not None):
        p.error("pass either --round N or --all")

    if not args.all:
        ingest(args.year, args.round, args.database_url, args.cache_dir)
        return

    failed = []
    for round_ in range(1, count_races(args.year, args.cache_dir) + 1):
        try:
            ingest(args.year, round_, args.database_url, args.cache_dir)
        except RuntimeError as e:  # a race that OpenF1 kept refusing: carry on, report at the end
            print(f"round {round_} failed: {e}")
            failed.append(round_)
    if failed:
        raise SystemExit(f"failed rounds: {failed} (run them again with --round)")


if __name__ == "__main__":
    main()
