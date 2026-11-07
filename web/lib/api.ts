// Server components talk to API_URL (inside Docker: http://api:8080); the browser to NEXT_PUBLIC_API_URL.
const PUBLIC_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
export const API_URL =
  typeof window === "undefined" ? (process.env.API_URL ?? PUBLIC_URL) : PUBLIC_URL;

export async function api<T>(path: string): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`API ${res.status} em ${path}`);
  return res.json() as Promise<T>;
}

export type Race = {
  id: number;
  year: number;
  round: number;
  name: string;
  circuit: string;
  country: string;
  event_date?: string;
  total_laps: number | null;
};

export type Driver = {
  code: string;
  number: string;
  name: string;
  team: string;
  team_colour: string; // hex without '#'
  headshot_url: string;
  laps: number;
  clean_laps: number;
  best_lap_id: number | null;
  best_lap_ms: number | null;
  consistency_s: number | null;
};

export type Lap = {
  id: number;
  driver: string;
  lap: number;
  lap_time_ms: number | null;
  compound: string;
  tyre_life: number | null;
  stint: number | null;
  position: number | null;
  clean: boolean;
  has_telemetry: boolean;
};

export type Stint = {
  driver: string;
  stint: number;
  compound: string;
  laps: number;
  slope_s_per_lap: number | null;
  points: { tyre_life: number; lap_time_s: number }[];
};

export type LapInfo = {
  id: number;
  driver: string;
  name: string;
  team: string;
  team_colour: string;
  headshot_url: string;
  lap: number;
  lap_time_ms: number | null;
};

export type LapSeries = LapInfo & {
  speed: number[];
  throttle: number[];
  brake: number[];
  gear: number[];
  rpm: number[];
};

export type Compare = {
  step_m: number;
  distance: number[];
  a: LapSeries;
  b: LapSeries;
  delta: number[];
  track: { x: number[]; y: number[] };
};

export type Gain = {
  corner: number;
  start_m: number;
  apex_m: number;
  end_m: number;
  time_gain_s: number;
  reason: "early_braking" | "low_min_speed" | "slow_exit" | "mixed";
  summary: string;
  brake_diff_m?: number;
  throttle_diff_m?: number;
  min_speed_a: number;
  min_speed_b: number;
};

export type Report = { a: LapInfo; b: LapInfo; total_delta_s: number; gains: Gain[] };
