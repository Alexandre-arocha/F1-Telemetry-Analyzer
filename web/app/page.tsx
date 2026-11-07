import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type Race } from "@/lib/api";

function fmtDate(iso?: string): string {
  if (!iso) return "";
  return new Date(`${iso}T12:00:00Z`).toLocaleDateString("pt-BR", { day: "numeric", month: "long", timeZone: "UTC" });
}

export default async function Home() {
  const races = await api<Race[]>("/api/races");
  const years = [...new Set(races.map((r) => r.year))];

  return (
    <div className="space-y-8">
      <div className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight">Telemetria de corridas de F1</h1>
        <p className="text-muted-foreground">
          Escolha um Grande Prêmio. Depois compare duas voltas (de qualquer piloto) e veja, curva a curva, onde uma
          delas perde tempo para a outra.
        </p>
      </div>
      {races.length === 0 ? (
        <p className="text-muted-foreground">
          Nenhuma corrida carregada ainda. Rode <code>make ingest YEAR=2024 ROUND=1</code>.
        </p>
      ) : (
        years.map((year) => (
          <section key={year} className="space-y-3">
            <h2 className="text-lg font-medium">
              Temporada {year} · {races.filter((r) => r.year === year).length} GPs
            </h2>
            <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {races
                .filter((r) => r.year === year)
                .map((r) => (
                  <li key={r.id}>
                    <Link href={`/race/${r.id}`}>
                      <Card className="h-full transition-colors hover:border-red-500">
                        <CardHeader>
                          <CardTitle className="flex items-start gap-3">
                            <span className="rounded-md bg-red-600 px-2 py-0.5 text-sm tabular-nums">{r.round}</span>
                            <span>{r.name}</span>
                          </CardTitle>
                        </CardHeader>
                        <CardContent className="text-sm text-muted-foreground">
                          {r.circuit}, {r.country} · {fmtDate(r.event_date)}
                        </CardContent>
                      </Card>
                    </Link>
                  </li>
                ))}
            </ul>
          </section>
        ))
      )}
    </div>
  );
}
