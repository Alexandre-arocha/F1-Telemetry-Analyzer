import Link from "next/link";
import { ReasonBadge } from "@/components/reason-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type Report } from "@/lib/api";
import { DriverAvatar } from "@/components/driver-avatar";
import { fmtGap, fmtLap, TIPS } from "@/lib/format";

export default async function ReportPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ a?: string; b?: string }>;
}) {
  const { id } = await params;
  const { a, b } = await searchParams;
  if (!a || !b) return <p>Escolha duas voltas na página da corrida.</p>;
  const report = await api<Report>(`/api/report?a=${encodeURIComponent(a)}&b=${encodeURIComponent(b)}`);

  return (
    <div className="space-y-6">
      <div>
        <Link href={`/race/${id}`} className="text-sm text-muted-foreground hover:underline">
          ← comparação
        </Link>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">Onde ganhar tempo</h1>
        <div className="mt-3 flex flex-wrap items-center gap-6">
          {[
            { l: report.b, role: "Volta a melhorar" },
            { l: report.a, role: "Referência" },
          ].map(({ l, role }) => (
            <div key={role} className="flex items-center gap-3">
              <DriverAvatar src={l.headshot_url} name={l.name} colour={l.team_colour} size={52} />
              <div className="text-sm leading-tight">
                <p className="text-muted-foreground">{role}</p>
                <p className="font-medium">
                  {l.name} · {l.team}
                </p>
                <p className="tabular-nums text-muted-foreground">
                  volta {l.lap} · {fmtLap(l.lap_time_ms)}
                </p>
              </div>
            </div>
          ))}
        </div>
        <p className="mt-3 text-sm text-muted-foreground">
          {report.b.driver} termina {fmtGap(report.total_delta_s)} de {report.a.driver}. Abaixo estão as 3 curvas onde
          {" "}{report.b.driver} mais perde tempo, ou seja, onde mais poderia ganhar.
        </p>
      </div>

      {report.gains.length === 0 ? (
        <p className="text-muted-foreground">
          A volta B não perde tempo para a referência em nenhuma curva detectada.
        </p>
      ) : (
        <ol className="grid gap-4 md:grid-cols-3">
          {report.gains.map((g, i) => (
            <li key={g.corner}>
              <Card className="h-full">
                <CardHeader className="space-y-2">
                  <CardTitle className="flex items-baseline justify-between">
                    <span>
                      {i + 1}º · Curva {g.corner}
                    </span>
                    <span className="text-xl tabular-nums text-emerald-400">−{g.time_gain_s.toFixed(3)} s</span>
                  </CardTitle>
                  <ReasonBadge reason={g.reason} />
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  <p>{g.summary}</p>
                  <p className="text-muted-foreground">{TIPS[g.reason]}</p>
                  <dl className="grid grid-cols-2 gap-x-3 gap-y-1 text-muted-foreground">
                    <dt>Trecho</dt>
                    <dd className="tabular-nums text-foreground">
                      {Math.round(g.start_m)}–{Math.round(g.end_m)} m
                    </dd>
                    <dt>Ápice</dt>
                    <dd className="tabular-nums text-foreground">{Math.round(g.apex_m)} m</dd>
                    <dt>Mín. A / B</dt>
                    <dd className="tabular-nums text-foreground">
                      {Math.round(g.min_speed_a)} / {Math.round(g.min_speed_b)} km/h
                    </dd>
                    {g.brake_diff_m != null && (
                      <>
                        <dt>Freada</dt>
                        <dd className="tabular-nums text-foreground">
                          {g.brake_diff_m >= 0 ? "+" : ""}
                          {Math.round(g.brake_diff_m)} m antes
                        </dd>
                      </>
                    )}
                    {g.throttle_diff_m != null && (
                      <>
                        <dt>Aceleração</dt>
                        <dd className="tabular-nums text-foreground">
                          {g.throttle_diff_m >= 0 ? "+" : ""}
                          {Math.round(g.throttle_diff_m)} m depois
                        </dd>
                      </>
                    )}
                  </dl>
                </CardContent>
              </Card>
            </li>
          ))}
        </ol>
      )}

      <p className="text-sm text-muted-foreground">
        Os ganhos são medidos na telemetria (delta de tempo no trecho). As causas são estimativas por comparação
        entre as duas voltas, não dados internos de equipe.
      </p>
    </div>
  );
}
