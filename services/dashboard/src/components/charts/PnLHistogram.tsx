"use client";

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";

interface HistogramBin {
  range: string;
  count: number;
}

interface PnLHistogramProps {
  pnls: (number | string)[];
  bins?: number;
  loading?: boolean;
}

function buildHistogram(pnls: number[], binCount: number): HistogramBin[] {
  if (pnls.length === 0) return [];

  // #2: Avoid Math.min/max spread on large arrays — use loop
  let min = Infinity;
  let max = -Infinity;
  for (const p of pnls) {
    if (p < min) min = p;
    if (p > max) max = p;
  }
  const range = max - min || 1;
  const binWidth = range / binCount;

  const bins: HistogramBin[] = [];
  for (let i = 0; i < binCount; i++) {
    const low = min + i * binWidth;
    const high = low + binWidth;
    bins.push({
      range: `$${low.toFixed(2)}-${high.toFixed(2)}`,
      count: 0,
    });
  }

  for (const pnl of pnls) {
    let idx = Math.floor((pnl - min) / binWidth);
    if (idx >= binCount) idx = binCount - 1;
    if (idx < 0) idx = 0;
    bins[idx].count++;
  }

  return bins;
}

export function PnLHistogram({ pnls, bins = 20, loading }: PnLHistogramProps) {
  if (loading) {
    return <div className="h-64 animate-pulse bg-white/5 rounded-xl" />;
  }

  if (!pnls || pnls.length === 0) {
    return <div className="h-64 flex items-center justify-center text-gray-400">No data</div>;
  }

  // Convert string[] to number[]
  const numericPnls = pnls.map((p) => typeof p === "string" ? parseFloat(p) : p).filter((n) => !isNaN(n));
  const histogramData = buildHistogram(numericPnls, bins);

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5">
      <h3 className="text-sm font-medium text-gray-400 mb-4">PnL Distribution</h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={histogramData}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
          <XAxis dataKey="range" stroke="#666" fontSize={9} angle={-45} textAnchor="end" height={60} />
          <YAxis stroke="#666" fontSize={11} />
          <Tooltip
            contentStyle={{ background: "#1a1f2e", border: "1px solid rgba(255,255,255,0.1)", borderRadius: "8px" }}
            labelStyle={{ color: "#fff" }}
            formatter={(value: number) => [value, "Trades"]}
          />
          <Bar dataKey="count" fill="#00d4ff" radius={[2, 2, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
