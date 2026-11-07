import { DriverAvatar } from "@/components/driver-avatar";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { Driver, Stint } from "@/lib/api";
import { fmtLap } from "@/lib/format";

export function DriverTable({ drivers }: { drivers: Driver[] }) {
  const rows = [...drivers].sort((a, b) => (a.best_lap_ms ?? 1e12) - (b.best_lap_ms ?? 1e12));
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Piloto</TableHead>
          <TableHead>Equipe</TableHead>
          <TableHead className="text-right">Melhor volta</TableHead>
          <TableHead className="text-right">Voltas limpas</TableHead>
          <TableHead className="text-right">Consistência (σ)</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((d) => (
          <TableRow key={d.code}>
            <TableCell>
              <span className="flex items-center gap-2 font-medium">
                <DriverAvatar src={d.headshot_url} name={d.name} colour={d.team_colour} size={32} />
                {d.name}
              </span>
            </TableCell>
            <TableCell>{d.team}</TableCell>
            <TableCell className="text-right tabular-nums">{fmtLap(d.best_lap_ms)}</TableCell>
            <TableCell className="text-right tabular-nums">
              {d.clean_laps}/{d.laps}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {d.consistency_s == null ? "—" : `${d.consistency_s.toFixed(3)} s`}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export function StintTable({ stints }: { stints: Stint[] }) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Piloto</TableHead>
          <TableHead>Stint</TableHead>
          <TableHead>Pneu</TableHead>
          <TableHead className="text-right">Voltas limpas</TableHead>
          <TableHead className="text-right">Degradação (s/volta)</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {stints.map((s) => (
          <TableRow key={`${s.driver}-${s.stint}`}>
            <TableCell className="font-medium">{s.driver}</TableCell>
            <TableCell>{s.stint}</TableCell>
            <TableCell>
              <Badge variant="outline">{s.compound || "?"}</Badge>
            </TableCell>
            <TableCell className="text-right tabular-nums">{s.laps}</TableCell>
            <TableCell className="text-right tabular-nums">
              {s.slope_s_per_lap == null ? "—" : `${s.slope_s_per_lap >= 0 ? "+" : ""}${s.slope_s_per_lap.toFixed(3)}`}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
