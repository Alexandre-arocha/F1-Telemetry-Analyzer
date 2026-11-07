export function fmtLap(ms: number | null | undefined): string {
  if (ms == null) return "—";
  const m = Math.floor(ms / 60000);
  const s = ((ms % 60000) / 1000).toFixed(3).padStart(6, "0");
  return `${m}:${s}`;
}

export function fmtGap(seconds: number): string {
  return `${seconds >= 0 ? "+" : "−"}${Math.abs(seconds).toFixed(3)} s`;
}

export const TIPS: Record<string, string> = {
  early_braking: "Dica: freie mais perto da curva, no mesmo ponto da referência.",
  low_min_speed: "Dica: carregue mais velocidade no meio da curva.",
  slow_exit: "Dica: volte ao acelerador mais cedo na saída da curva.",
  mixed: "Sem uma causa única: o tempo se perde aos poucos ao longo do trecho.",
};

export const REASONS: Record<string, string> = {
  early_braking: "Frenagem antecipada",
  low_min_speed: "Velocidade mínima menor",
  slow_exit: "Saída de curva lenta",
  mixed: "Perda distribuída",
};
