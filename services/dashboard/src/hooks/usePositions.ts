"use client";

import { useEffect, useRef, useState } from "react";
import { fetchPositions } from "@/lib/api";
import { useWSContext } from "@/lib/ws-context";
import type { Position } from "@/types";

interface UsePositionsResult {
  data: Position[];
  loading: boolean;
  error: string | null;
  wsStatus: "connecting" | "connected" | "disconnected";
}

export function usePositions(): UsePositionsResult {
  const { positionData, wsStatus } = useWSContext();
  const [data, setData] = useState<Position[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const wsDataReceived = useRef(false);

  useEffect(() => {
    if (positionData && !wsDataReceived.current) {
      wsDataReceived.current = true;
      setData(positionData);
      setLoading(false);
    }
  }, [positionData]);

  useEffect(() => {
    let cancelled = false;

    fetchPositions()
      .then((positions) => {
        if (!cancelled && !wsDataReceived.current) {
          setData(positions);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load positions");
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  return { data, loading, error, wsStatus };
}
