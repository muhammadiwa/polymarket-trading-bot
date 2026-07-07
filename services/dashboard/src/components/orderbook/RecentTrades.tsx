"use client";

interface Trade {
  price: string;
  size: string;
  side: string;
  timestamp: string;
}

interface RecentTradesProps {
  trades: Trade[];
}

export function RecentTrades({ trades }: RecentTradesProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md overflow-hidden">
      <div className="px-4 py-3 border-b border-white/10">
        <h3 className="text-sm font-medium text-gray-400">Recent Trades</h3>
      </div>
      <div className="px-4 py-2 grid grid-cols-4 gap-2 text-xs text-gray-400 border-b border-white/5">
        <span>Price</span><span>Size</span><span>Side</span><span className="text-right">Time</span>
      </div>
      <div className="max-h-[300px] overflow-y-auto">
        {trades.length === 0 ? (
          <div className="px-4 py-8 text-center text-gray-500 text-sm">No trades</div>
        ) : (
          trades.slice(0, 100).map((trade, i) => (
            <div key={i} className="px-4 py-1 grid grid-cols-4 gap-2 text-xs">
              <span className="font-mono text-white">{parseFloat(trade.price).toFixed(4)}</span>
              <span className="font-mono text-gray-300">{parseFloat(trade.size).toFixed(2)}</span>
              <span className={trade.side === "BUY" ? "text-[#00ff88]" : "text-[#ff4757]"}>
                {trade.side}
              </span>
              <span className="font-mono text-gray-400 text-right">
                {trade.timestamp ? new Date(trade.timestamp).toLocaleTimeString() : "—"}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
