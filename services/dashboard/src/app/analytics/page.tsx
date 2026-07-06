"use client";

import { useState } from "react";
import { useAnalytics } from "@/hooks/useAnalytics";
import { PnLLineChart } from "@/components/charts/PnLLineChart";
import { PnLHistogram } from "@/components/charts/PnLHistogram";
import { StrategyPieChart } from "@/components/charts/StrategyPieChart";
import { downloadCSV } from "@/lib/api";

export default function AnalyticsPage() {
  const [startDate, setStartDate] = useState(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 1);
    return d.toISOString().split("T")[0];
  });
  const [endDate, setEndDate] = useState(() => new Date().toISOString().split("T")[0]);
  const [exporting, setExporting] = useState(false);

  const { pnlData, histogramData, loading, error } = useAnalytics(startDate, endDate);

  const handleExport = async () => {
    setExporting(true);
    try {
      await downloadCSV(startDate, endDate);
    } catch (err) {
      console.error("Export failed:", err);
    } finally {
      setExporting(false);
    }
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">Analytics</h1>
        <div className="flex items-center gap-3">
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
          />
          <span className="text-gray-400">to</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
          />
          <button
            onClick={handleExport}
            disabled={exporting}
            className="px-4 py-2 rounded-lg bg-[#00d4ff] text-black font-medium text-sm hover:bg-[#00d4ff]/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {exporting ? "Exporting..." : "Export CSV"}
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 backdrop-blur-md p-5 text-[#ff4757]" role="alert">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 gap-6">
        <PnLLineChart data={pnlData?.by_period?.map(p => ({ ...p, pnl: parseFloat(p.pnl), trade_count: p.trade_count })) ?? []} loading={loading} />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <PnLHistogram pnls={histogramData?.pnls ?? []} bins={histogramData?.bins ?? 20} loading={loading} />
          <StrategyPieChart data={pnlData?.by_strategy ?? []} loading={loading} />
        </div>
      </div>
    </div>
  );
}
