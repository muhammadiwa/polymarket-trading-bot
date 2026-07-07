"use client";

interface Decision {
  market_id: string;
  detected: string;
  decision: string;
  reason: string;
  score: string;
  risk_result: string;
  timestamp: string;
}

interface DecisionLogProps {
  decisions: Decision[];
}

export function DecisionLog({ decisions }: DecisionLogProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md overflow-hidden">
      <div className="px-4 py-3 border-b border-white/10">
        <h3 className="text-sm font-medium text-gray-400">Decision Log</h3>
      </div>
      <div className="px-4 py-2 grid grid-cols-6 gap-2 text-xs text-gray-400 border-b border-white/5">
        <span>Time</span><span>Market</span><span>Detected</span><span>Decision</span><span>Reason</span><span>Risk</span>
      </div>
      <div className="max-h-[400px] overflow-y-auto">
        {decisions.length === 0 ? (
          <div className="px-4 py-8 text-center text-gray-500 text-sm">No decisions yet</div>
        ) : (
          decisions.map((d, i) => (
            <div key={i} className="px-4 py-2 grid grid-cols-6 gap-2 text-xs border-b border-white/5">
              <span className="font-mono text-gray-400">
                {d.timestamp ? new Date(d.timestamp).toLocaleTimeString() : "—"}
              </span>
              <span className="text-white truncate" title={d.market_id}>
                {d.market_id.slice(0, 12)}...
              </span>
              <span className="text-gray-300">{d.detected}</span>
              <span className={`font-medium ${
                d.decision === "EXECUTE" ? "text-[#00ff88]" : d.decision === "FILTER" ? "text-gray-400" : "text-yellow-400"
              }`}>
                {d.decision}
              </span>
              <span className="text-gray-400 truncate" title={d.reason}>{d.reason}</span>
              <span className={`font-medium ${d.risk_result === "ALLOWED" ? "text-[#00ff88]" : "text-[#ff4757]"}`}>
                {d.risk_result}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
