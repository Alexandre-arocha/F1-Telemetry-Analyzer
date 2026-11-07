"use client";

import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export type ChartLine = { key: string; name: string; color: string; step?: boolean };

export function TraceChart({
  title,
  data,
  lines,
  zeroLine,
}: {
  title: string;
  data: Record<string, number>[];
  lines: ChartLine[];
  zeroLine?: boolean;
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-52">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} syncId="lap" margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#ffffff22" />
            <XAxis
              dataKey="d"
              type="number"
              domain={[0, "dataMax"]}
              tickFormatter={(v) => `${v} m`}
              fontSize={11}
            />
            <YAxis fontSize={11} width={44} />
            <Tooltip
              labelFormatter={(v) => `${v} m`}
              contentStyle={{ background: "#111", border: "1px solid #333", fontSize: 12 }}
            />
            {zeroLine && <ReferenceLine y={0} stroke="#888" />}
            {lines.map((l) => (
              <Line
                key={l.key}
                dataKey={l.key}
                name={l.name}
                stroke={l.color}
                dot={false}
                strokeWidth={1.5}
                type={l.step ? "stepAfter" : "linear"}
                isAnimationActive={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
