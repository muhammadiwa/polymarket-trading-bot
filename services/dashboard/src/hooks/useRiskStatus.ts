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

  // WS effect — always apply (not just first message)
  useEffect(() => {
    if (riskData) {
      wsDataReceived.current = true;
      setData(riskData);
      setLoading(false);
      setError(null);
    }
  }, [riskData]);

  // REST fallback — only if WS hasn't sent data yet
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
        if (!cancelled && !wsDataReceived.current) {
          setError(err instanceof Error ? err.message : "Failed to load risk status");
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  // #7: refresh with generation counter to prevent stale overwrites
  const refreshGen = useRef(0);
  const refresh = async () => {
    const gen = ++refreshGen.current;
    try {
      const status = await fetchRiskStatus();
      if (gen === refreshGen.current) {
        setData(status);
        setError(null);
      }
    } catch (err) {
      if (gen === refreshGen.current) {
        setError(err instanceof Error ? err.message : "Failed to refresh risk status");
      }
    }
  };

  return { data, loading, error, wsStatus, refresh };
}
