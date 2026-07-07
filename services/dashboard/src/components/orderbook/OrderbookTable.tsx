"use client";

interface OrderbookLevel {
  price: string;
  size: string;
  cumulative: string;
}

interface OrderbookTableProps {
  bids: OrderbookLevel[];
  asks: OrderbookLevel[];
  spread: string;
}

export function OrderbookTable({ bids, asks, spread }: OrderbookTableProps) {
  // #11: Use reduce instead of spread to avoid stack overflow
  const maxSize = Math.max(
    bids.reduce((m, b) => Math.max(m, parseFloat(b.size) || 0), 0),
    asks.reduce((m, a) => Math.max(m, parseFloat(a.size) || 0), 0),
    1
  );

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md overflow-hidden">
      <div className="px-4 py-3 border-b border-white/10 flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-400">Orderbook</h3>
        <span className="text-xs text-gray-400">Spread: <span className="text-white font-mono">{spread}</span></span>
      </div>

      <div className="grid grid-cols-2 gap-0">
        {/* Bids (left) */}
        <div className="border-r border-white/10">
          <div className="px-4 py-2 grid grid-cols-3 gap-2 text-xs text-gray-400 border-b border-white/5">
            <span>Price</span><span className="text-right">Size</span><span className="text-right">Total</span>
          </div>
          <div className="max-h-[300px] overflow-y-auto">
            {bids.slice(0, 20).map((bid) => (
              <div key={`${bid.price}-${bid.size}`} className="relative px-4 py-1 grid grid-cols-3 gap-2 text-xs">
                <div className="absolute inset-y-0 right-0 bg-[#00ff88]/5" style={{ width: `${((parseFloat(bid.size) || 0) / maxSize) * 100}%` }} />
                <span className="text-[#00ff88] font-mono relative">{bid.price}</span>
                <span className="text-gray-300 font-mono text-right relative">{parseFloat(bid.size).toFixed(2)}</span>
                <span className="text-gray-400 font-mono text-right relative">{parseFloat(bid.cumulative).toFixed(2)}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Asks (right) */}
        <div>
          <div className="px-4 py-2 grid grid-cols-3 gap-2 text-xs text-gray-400 border-b border-white/5">
            <span>Price</span><span className="text-right">Size</span><span className="text-right">Total</span>
          </div>
          <div className="max-h-[300px] overflow-y-auto">
            {asks.slice(0, 20).map((ask) => (
              <div key={`${ask.price}-${ask.size}`} className="relative px-4 py-1 grid grid-cols-3 gap-2 text-xs">
                <div className="absolute inset-y-0 left-0 bg-[#ff4757]/5" style={{ width: `${((parseFloat(ask.size) || 0) / maxSize) * 100}%` }} />
                <span className="text-[#ff4757] font-mono relative">{ask.price}</span>
                <span className="text-gray-300 font-mono text-right relative">{parseFloat(ask.size).toFixed(2)}</span>
                <span className="text-gray-400 font-mono text-right relative">{parseFloat(ask.cumulative).toFixed(2)}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
