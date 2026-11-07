"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { buttonVariants } from "@/components/ui/button";
import { LapPicker } from "@/components/lap-picker";
import { TraceChart } from "@/components/trace-chart";
import { API_URL, type Compare, type Driver, type Lap } from "@/lib/api";
import { fmtGap, fmtLap } from "@/lib/format";

const COLOR_A = "#22d3ee";
const COLOR_B = "#fb923c";

export function CompareView({
  raceId,
  drivers,
  laps,
}: {
  raceId: number;
  drivers: Driver[];
  laps: Lap[]; // only laps with telemetry
}) {
  const ranked = useMemo(
    () =>
      drivers
        .filter((d) => d.best_lap_id && d.best_lap_ms)
        .sort((x, y) => x.best_lap_ms! - y.best_lap_ms!),
    [drivers],
  );
  const [a, setA] = useState<number | null>(ranked[0]?.best_lap_id ?? null);
  const [b, setB] = useState<number | null>(ranked[1]?.best_lap_id ?? null);
  const [cmp, setCmp] = useState<Compare | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (a == null || b == null) return;
    const ctrl = new AbortController();
    setError(null);
    fetch(`${API_URL}/api/compare?a=${a}&b=${b}`, { signal: ctrl.signal })
      .then((r) => {
        if (!r.ok) throw new Error(`API ${r.status}`);
        return r.json() as Promise<Compare>;
      })
      .then(setCmp)
      .catch((e: Error) => e.name !== "AbortError" && setError(e.message));
    return () => ctrl.abort();
  }, [a, b]);

  const rows = useMemo(() => {
    if (!cmp) return null;
    return cmp.distance.map((d, i) => ({
      d,
      sa: cmp.a.speed[i],
      sb: cmp.b.speed[i],
      ba: cmp.a.brake[i],
      bb: cmp.b.brake[i],
      ta: cmp.a.throttle[i],
      tb: cmp.b.throttle[i],
      ga: cmp.a.gear[i],
      gb: cmp.b.gear[i],
      delta: cmp.delta[i],
    }));
  }, [cmp]);

  if (ranked.length < 2) {
    return <p className="text-muted-foreground">Esta corrida não tem voltas com telemetria suficientes.</p>;
  }

  const nameA = cmp ? `${cmp.a.driver} V${cmp.a.lap}` : "A";
  const nameB = cmp ? `${cmp.b.driver} V${cmp.b.lap}` : "B";
  const pair = (suffix: string, step = false) => [
    { key: `${suffix}a`, name: nameA, color: COLOR_A, step },
    { key: `${suffix}b`, name: nameB, color: COLOR_B, step },
  ];

  return (
    <section className="space-y-4">
      <div className="space-y-2">
        <LapPicker label="A · referência" color={COLOR_A} drivers={drivers} laps={laps} lapId={a} onChange={setA} />
        <LapPicker label="B · a melhorar" color={COLOR_B} drivers={drivers} laps={laps} lapId={b} onChange={setB} />
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        {cmp && (
          <span className="text-muted-foreground">
            {fmtLap(cmp.a.lap_time_ms)} contra {fmtLap(cmp.b.lap_time_ms)} · B termina{" "}
            <b className="text-foreground">{fmtGap(cmp.delta[cmp.delta.length - 1])}</b> de A
          </span>
        )}
        {a != null && b != null && (
          <Link
            href={`/race/${raceId}/report?a=${a}&b=${b}`}
            className={buttonVariants({ size: "sm" })}
          >
            Ver relatório de ganhos
          </Link>
        )}
      </div>

      {error && <p className="text-red-400">Falha ao carregar a comparação: {error}</p>}
      {!rows && !error && <p className="text-muted-foreground">Carregando telemetria…</p>}
      {rows && (
        <div className="grid gap-4 lg:grid-cols-2">
          <TraceChart title="Velocidade (km/h)" data={rows} lines={pair("s")} />
          <TraceChart title="Delta acumulado (s) · positivo = B atrás" data={rows} lines={[{ key: "delta", name: "B − A", color: "#a78bfa" }]} zeroLine />
          <TraceChart title="Freio (0/1)" data={rows} lines={pair("b", true)} />
          <TraceChart title="Acelerador (%)" data={rows} lines={pair("t")} />
          <TraceChart title="Marcha" data={rows} lines={pair("g", true)} />
        </div>
      )}
    </section>
  );
}
