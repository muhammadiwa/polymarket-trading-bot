"use client";

import { useEffect, useRef, useState } from "react";
import { fetchSystemHealth } from "@/lib/api";
import { useWSContext } from "@/lib/ws-context";
import type { SystemHealth } from "@/types";

interface UseSystemHealthResult {
  data: SystemHealth | null;
  loading: boolean;
  error: string | null;
  wsStatus: "connecting" | "connected" | "disconnected";
  refresh: () => Promise<void>;
}

export function useSystemHealth(): UseSystemHealthResult {
  const { healthData, wsStatus } = useWSContext();
  const [data, setData] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const wsDataReceived = useRef(false);

  // #15: Reset wsDataReceived on WS disconnect
  useEffect(() => {
    if (wsStatus === "disconnected") {
      wsDataReceived.current = false;
    }
  }, [wsStatus]);

  useEffect(() => {
    if (healthData && !wsDataReceived.current) {
      wsDataReceived.current = true;
      setData(healthData);
      setLoading(false);
    }
  }, [healthData]);

  // #17: Initial fetch + 5s polling fallback
  useEffect(() => {
    let cancelled = false;
    let intervalId: ReturnType<typeof setInterval> | null = null;

    const poll = async () => {
      try {
        const health = await fetchSystemHealth();
        if (!cancelled && !wsDataReceived.current) {
          setData(health);
          setLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load system health");
          setLoading(false);
        }
      }
    };

    poll();
    intervalId = setInterval(poll, 5000);

    return () => {
      cancelled = true;
      if (intervalId) clearInterval(intervalId);
    };
  }, []);

  const refresh = async () => {
    try {
      const health = await fetchSystemHealth();
      setData(health);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to refresh system health");
    }
  };

  return { data, loading, error, wsStatus, refresh };
}
