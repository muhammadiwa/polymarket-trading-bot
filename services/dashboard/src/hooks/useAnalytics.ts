"use client";

import { useCallback, useEffect, useRef, useState } from "react";
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
  const requestIdRef = useRef(0);

  const fetchData = useCallback(async () => {
    if (!startDate || !endDate) return;
    const requestId = ++requestIdRef.current;
    setLoading(true);
    setError(null);
    try {
      const [pnl, hist] = await Promise.all([
        fetchAnalyticsPnL(startDate, endDate, "day"),
        fetchAnalyticsHistogram(startDate, endDate),
      ]);
      // #1: Discard stale responses
      if (requestId !== requestIdRef.current) return;
      setPnlData(pnl);
      setHistogramData(hist);
    } catch (err) {
      if (requestId !== requestIdRef.current) return;
      setError(err instanceof Error ? err.message : "Failed to load analytics");
    } finally {
      if (requestId === requestIdRef.current) setLoading(false);
    }
  }, [startDate, endDate]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { pnlData, histogramData, loading, error, refresh: fetchData };
}
