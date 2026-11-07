import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type Race } from "@/lib/api";

export default async function Home() {
  const races = await api<Race[]>("/api/races");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Telemetria de corridas</h1>
        <p className="text-muted-foreground">
          Escolha uma corrida, compare duas voltas por distância percorrida e veja os três maiores pontos
          de ganho de tempo.
        </p>
      </div>
      {races.length === 0 ? (
        <p className="text-muted-foreground">
          Nenhuma corrida carregada ainda. Rode <code>make ingest YEAR=2024 ROUND=1</code>.
        </p>
      ) : (
        <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {races.map((r) => (
            <li key={r.id}>
              <Link href={`/race/${r.id}`}>
                <Card className="transition-colors hover:border-red-500">
                  <CardHeader>
                    <CardTitle>
                      {r.name} {r.year}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="text-sm text-muted-foreground">
                    Etapa {r.round} · {r.circuit}, {r.country}
                  </CardContent>
                </Card>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
