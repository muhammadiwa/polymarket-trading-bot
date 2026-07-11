"use client";

import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from "recharts";

interface StrategyData {
  strategy_id: string;
  total_pnl: string;
  trade_count: number;
}

interface StrategyPieChartProps {
  data: StrategyData[];
  loading?: boolean;
}

const COLORS = ["#00d4ff", "#00ff88", "#ff4757", "#ffd700", "#ff6b6b", "#4ecdc4", "#45b7d1", "#96ceb4"];

export function StrategyPieChart({ data, loading }: StrategyPieChartProps) {
  if (loading) {
    return <div className="h-64 animate-pulse bg-white/5 rounded-xl" />;
  }

  if (!data || data.length === 0) {
    return <div className="h-64 flex items-center justify-center text-gray-400">No data</div>;
  }

  const chartData = data
    .filter((d) => parseFloat(d.total_pnl) !== 0) // #4: Filter out zero-PnL strategies
    .map((d) => ({
      name: d.strategy_id,
      value: Math.abs(parseFloat(d.total_pnl)),
      pnl: parseFloat(d.total_pnl),
      trades: d.trade_count,
      isNegative: parseFloat(d.total_pnl) < 0,
    }));

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5">
      <h3 className="text-sm font-medium text-gray-400 mb-4">PnL by Strategy</h3>
      <ResponsiveContainer width="100%" height={300}>
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            outerRadius={100}
            innerRadius={50}
            dataKey="value"
            label={({ name, percent }) => `${name} (${(percent * 100).toFixed(1)}%)`}
          >
            {chartData.map((entry, index) => (
              <Cell key={entry.name} fill={COLORS[index % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{ background: "#1a1f2e", border: "1px solid rgba(255,255,255,0.1)", borderRadius: "8px" }}
            formatter={(value: number, name: string, props: { payload?: { pnl?: number; trades?: number } }) => {
              const pnl = props.payload?.pnl ?? 0;
              const trades = props.payload?.trades ?? 0;
              return [`$${pnl.toFixed(4)} (${trades} trades)`, name];
            }}
          />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
