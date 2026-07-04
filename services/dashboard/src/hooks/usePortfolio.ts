"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { fetchPortfolioOverview } from "@/lib/api";
import { useWSContext } from "@/lib/ws-context";
import type { PortfolioOverview } from "@/types";

interface UsePortfolioResult {
  data: PortfolioOverview | null;
  loading: boolean;
  error: string | null;
  wsStatus: "connecting" | "connected" | "disconnected";
}

export function usePortfolio(): UsePortfolioResult {
  const { portfolioData, wsStatus } = useWSContext();
  const [data, setData] = useState<PortfolioOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const wsDataReceived = useRef(false);

  useEffect(() => {
    if (portfolioData && !wsDataReceived.current) {
      wsDataReceived.current = true;
      setData(portfolioData);
      setLoading(false);
    }
  }, [portfolioData]);

  useEffect(() => {
    let cancelled = false;

    fetchPortfolioOverview()
      .then((overview) => {
        if (!cancelled && !wsDataReceived.current) {
          setData(overview);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load portfolio");
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  return { data, loading, error, wsStatus };
}
