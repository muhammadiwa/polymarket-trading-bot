"use client";

import { useMemo } from "react";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";

interface OrderbookLevel {
  price: string;
  size: string;
  cumulative: string;
}

interface DepthChartProps {
  bids: OrderbookLevel[];
  asks: OrderbookLevel[];
}

export function DepthChart({ bids, asks }: DepthChartProps) {
  if (bids.length === 0 && asks.length === 0) {
    return <div className="h-48 flex items-center justify-center text-gray-400 text-sm">No data</div>;
  }

  // #18: Memoize data transformation to prevent unnecessary re-renders
  const data = useMemo(() => {
    const d: { price: number; bidDepth: number; askDepth: number }[] = [];
    for (const bid of bids) {
      d.push({ price: parseFloat(bid.price), bidDepth: parseFloat(bid.cumulative), askDepth: 0 });
    }
    for (const ask of asks) {
      d.push({ price: parseFloat(ask.price), bidDepth: 0, askDepth: parseFloat(ask.cumulative) });
    }
    d.sort((a, b) => a.price - b.price);
    return d;
  }, [bids, asks]);

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5">
      <h3 className="text-sm font-medium text-gray-400 mb-4">Depth Chart</h3>
      <ResponsiveContainer width="100%" height={200}>
        <AreaChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
          <XAxis dataKey="price" stroke="#666" fontSize={10} tickFormatter={(v) => v.toFixed(2)} />
          <YAxis stroke="#666" fontSize={10} />
          <Tooltip
            contentStyle={{ background: "#1a1f2e", border: "1px solid rgba(255,255,255,0.1)", borderRadius: "8px" }}
            labelFormatter={(v) => `Price: ${v}`}
          />
          <Area type="stepAfter" dataKey="bidDepth" stroke="#00ff88" fill="#00ff88" fillOpacity={0.1} name="Bids" />
          <Area type="stepAfter" dataKey="askDepth" stroke="#ff4757" fill="#ff4757" fillOpacity={0.1} name="Asks" />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
