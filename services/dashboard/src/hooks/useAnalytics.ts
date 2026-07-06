"use client";

import { useCallback, useEffect, useState } from "react";
import { fetchAnalyticsPnL, fetchAnalyticsHistogram } from "@/lib/api";
import type { PnLData, HistogramData } from "@/types";

interface UseAnalyticsResult {
  pnlData: PnLData | null;
  histogramData: HistogramData | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useAnalytics(startDate: string, endDate: string): UseAnalyticsResult {
  const [pnlData, setPnlData] = useState<PnLData | null>(null);
  const [histogramData, setHistogramData] = useState<HistogramData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    if (!startDate || !endDate) return;
    setLoading(true);
    setError(null);
    try {
      const [pnl, hist] = await Promise.all([
        fetchAnalyticsPnL(startDate, endDate, "day"),
        fetchAnalyticsHistogram(startDate, endDate),
      ]);
      setPnlData(pnl);
      setHistogramData(hist);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load analytics");
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { pnlData, histogramData, loading, error, refresh: fetchData };
}
