"use client";

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Brush } from "recharts";

interface PnLDataPoint {
  date: string;
  pnl: number;
  trade_count: number;
}

interface PnLLineChartProps {
  data: PnLDataPoint[];
  loading?: boolean;
}

export function PnLLineChart({ data, loading }: PnLLineChartProps) {
  if (loading) {
    return <div className="h-64 animate-pulse bg-white/5 rounded-xl" />;
  }

  if (!data || data.length === 0) {
    return <div className="h-64 flex items-center justify-center text-gray-400">No data</div>;
  }

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5">
      <h3 className="text-sm font-medium text-gray-400 mb-4">PnL Over Time</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
          <XAxis dataKey="date" stroke="#666" fontSize={11} />
          <YAxis stroke="#666" fontSize={11} tickFormatter={(v) => `$${v.toFixed(2)}`} />
          <Tooltip
            contentStyle={{ background: "#1a1f2e", border: "1px solid rgba(255,255,255,0.1)", borderRadius: "8px" }}
            labelStyle={{ color: "#fff" }}
            formatter={(value: number) => [`$${value.toFixed(4)}`, "PnL"]}
          />
          <Line
            type="monotone"
            dataKey="pnl"
            stroke="#00d4ff"
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4 }}
          />
          <Brush dataKey="date" height={30} stroke="#00d4ff" fill="#0a0e17" />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
