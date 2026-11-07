export function fmtLap(ms: number | null | undefined): string {
  if (ms == null) return "—";
  const m = Math.floor(ms / 60000);
  const s = ((ms % 60000) / 1000).toFixed(3).padStart(6, "0");
  return `${m}:${s}`;
}

export function fmtGap(seconds: number): string {
  return `${seconds >= 0 ? "+" : "−"}${Math.abs(seconds).toFixed(3)} s`;
}

export const REASONS: Record<string, string> = {
  early_braking: "Frenagem antecipada",
  low_min_speed: "Velocidade mínima menor",
  slow_exit: "Saída de curva lenta",
  mixed: "Perda distribuída",
};
