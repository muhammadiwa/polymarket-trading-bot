"use client";

import { useEffect, useRef, useState } from "react";
import { fetchSystemHealth } from "@/lib/api";
import { useWSContext } from "@/lib/ws-context";
import type { WSStatus } from "@/lib/websocket";
import type { SystemHealth } from "@/types";

interface UseSystemHealthResult {
  data: SystemHealth | null;
  loading: boolean;
  error: string | null;
  wsStatus: WSStatus;
  refresh: () => Promise<void>;
}

export function useSystemHealth(): UseSystemHealthResult {
  const { healthData, wsStatus } = useWSContext();
  const [data, setData] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const wsDataReceived = useRef(false);

  // Reset wsDataReceived on WS disconnect
  useEffect(() => {
    if (wsStatus === "disconnected") {
      wsDataReceived.current = false;
    }
  }, [wsStatus]);

  // WS effect — always apply
  useEffect(() => {
    if (healthData) {
      wsDataReceived.current = true;
      setData(healthData);
      setLoading(false);
      setError(null);
    }
  }, [healthData]);

  // #2: Polling only when WS is NOT connected
  useEffect(() => {
    if (wsStatus === "connected") return; // WS is authoritative

    let cancelled = false;
    let intervalId: ReturnType<typeof setInterval> | null = null;

    const poll = async () => {
      try {
        const health = await fetchSystemHealth();
        if (!cancelled && !wsDataReceived.current) {
          setData(health);
          setLoading(false);
          setError(null); // #8: Clear error on success
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
  }, [wsStatus]);

  // #6: refresh with WS guard
  const refresh = async () => {
    try {
      const health = await fetchSystemHealth();
      if (!wsDataReceived.current) {
        setData(health);
      }
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to refresh system health");
    }
  };

  return { data, loading, error, wsStatus, refresh };
}
