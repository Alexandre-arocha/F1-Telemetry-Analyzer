import Link from "next/link";
import { notFound } from "next/navigation";
import { CompareView } from "@/components/compare-view";
import { DriverTable, StintTable } from "@/components/stats-tables";
import { api, type Driver, type Lap, type Race, type Stint } from "@/lib/api";

export default async function RacePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [races, drivers, laps, stints] = await Promise.all([
    api<Race[]>("/api/races"),
    api<Driver[]>(`/api/races/${id}/drivers`),
    api<Lap[]>(`/api/races/${id}/laps`),
    api<Stint[]>(`/api/races/${id}/degradation`),
  ]);
  const race = races.find((r) => String(r.id) === id);
  if (!race) notFound();

  return (
    <div className="space-y-10">
      <div>
        <Link href="/" className="text-sm text-muted-foreground hover:underline">
          ← corridas
        </Link>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">
          {race.name} {race.year}
        </h1>
        <p className="text-sm text-muted-foreground">
          {race.circuit}, {race.country} · {race.total_laps ?? "?"} voltas
        </p>
      </div>

      <section className="space-y-3">
        <h2 className="text-lg font-medium">Comparar duas voltas</h2>
        <CompareView raceId={race.id} drivers={drivers} laps={laps.filter((l) => l.has_telemetry && l.lap_time_ms)} />
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-medium">Pilotos e consistência</h2>
        <p className="text-sm text-muted-foreground">
          σ = desvio padrão dos tempos das voltas limpas (sem box, sem primeira volta, até 107% da mediana).
        </p>
        <DriverTable drivers={drivers} />
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-medium">Degradação de pneu por stint</h2>
        <p className="text-sm text-muted-foreground">
          Inclinação do tempo de volta pela idade do pneu, em voltas limpas. Inclui o efeito do combustível
          queimado, então valores perto de zero ou negativos são normais.
        </p>
        <StintTable stints={stints.filter((s) => s.slope_s_per_lap != null)} />
      </section>
    </div>
  );
}
