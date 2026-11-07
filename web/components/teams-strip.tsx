import { DriverAvatar } from "@/components/driver-avatar";
import type { Driver } from "@/lib/api";
import { fmtLap } from "@/lib/format";

// One card per team, in the team colour, with its drivers' photos and best laps.
export function TeamsStrip({ drivers }: { drivers: Driver[] }) {
  const teams = new Map<string, Driver[]>();
  for (const d of drivers) teams.set(d.team, [...(teams.get(d.team) ?? []), d]);
  const ordered = [...teams.entries()].sort(
    (a, b) => Math.min(...a[1].map((d) => d.best_lap_ms ?? 1e12)) - Math.min(...b[1].map((d) => d.best_lap_ms ?? 1e12)),
  );

  return (
    <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {ordered.map(([team, members]) => (
        <li
          key={team}
          className="rounded-xl border bg-card p-3"
          style={{ borderLeft: `6px solid #${members[0].team_colour || "555"}` }}
        >
          <p className="mb-2 font-medium">{team}</p>
          <div className="flex gap-4">
            {members.map((d) => (
              <div key={d.code} className="flex items-center gap-2">
                <DriverAvatar src={d.headshot_url} name={d.name} colour={d.team_colour} size={44} />
                <div className="text-sm leading-tight">
                  <p>{d.name}</p>
                  <p className="text-muted-foreground tabular-nums">{fmtLap(d.best_lap_ms)}</p>
                </div>
              </div>
            ))}
          </div>
        </li>
      ))}
    </ul>
  );
}
