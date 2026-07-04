"use client";

import { useEffect, useRef, useState } from "react";
import { fetchRiskStatus } from "@/lib/api";
import { useWSContext } from "@/lib/ws-context";
import type { RiskStatus } from "@/types";

interface UseRiskStatusResult {
  data: RiskStatus | null;
  loading: boolean;
  error: string | null;
  wsStatus: "connecting" | "connected" | "disconnected";
  refresh: () => Promise<void>;
}

export function useRiskStatus(): UseRiskStatusResult {
  const { riskData, wsStatus } = useWSContext();
  const [data, setData] = useState<RiskStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const wsDataReceived = useRef(false);

  useEffect(() => {
    if (riskData && !wsDataReceived.current) {
      wsDataReceived.current = true;
      setData(riskData);
      setLoading(false);
    }
  }, [riskData]);

  useEffect(() => {
    let cancelled = false;

    fetchRiskStatus()
      .then((status) => {
        if (!cancelled && !wsDataReceived.current) {
          setData(status);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load risk status");
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const refresh = async () => {
    try {
      const status = await fetchRiskStatus();
      setData(status);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to refresh risk status");
    }
  };

  return { data, loading, error, wsStatus, refresh };
}
