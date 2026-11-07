"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Driver, Lap } from "@/lib/api";
import { fmtLap } from "@/lib/format";

export function LapPicker({
  label,
  color,
  drivers,
  laps,
  lapId,
  onChange,
}: {
  label: string;
  color: string;
  drivers: Driver[];
  laps: Lap[]; // only laps with telemetry
  lapId: number | null;
  onChange: (lapId: number) => void;
}) {
  const driver = laps.find((l) => l.id === lapId)?.driver ?? "";
  const driverLaps = laps.filter((l) => l.driver === driver);

  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="w-32 text-sm font-medium" style={{ color }}>
        {label}
      </span>
      <Select
        value={driver}
        onValueChange={(code) => {
          const best = drivers.find((d) => d.code === code)?.best_lap_id;
          if (best) onChange(best);
        }}
      >
        <SelectTrigger className="w-44">
          <SelectValue placeholder="Piloto" />
        </SelectTrigger>
        <SelectContent>
          {drivers
            .filter((d) => d.best_lap_id)
            .map((d) => (
              <SelectItem key={d.code} value={d.code}>
                {d.code} · {d.team}
              </SelectItem>
            ))}
        </SelectContent>
      </Select>
      <Select value={lapId ? String(lapId) : ""} onValueChange={(v) => onChange(Number(v))}>
        <SelectTrigger className="w-72">
          <SelectValue placeholder="Volta" />
        </SelectTrigger>
        <SelectContent>
          {driverLaps.map((l) => (
            <SelectItem key={l.id} value={String(l.id)}>
              Volta {l.lap} · {fmtLap(l.lap_time_ms)} · {l.compound || "?"}
              {l.clean ? "" : " · não limpa"}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
