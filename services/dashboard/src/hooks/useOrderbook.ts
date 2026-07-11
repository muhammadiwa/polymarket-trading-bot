"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { fetchOrderbook, fetchRecentTrades } from "@/lib/api";
import type { OrderbookLevel, OrderbookSnapshot, RecentTrade } from "@/types";

interface UseOrderbookResult {
  orderbook: OrderbookSnapshot | null;
  trades: RecentTrade[];
  loading: boolean;
  error: string | null;
}

export function useOrderbook(marketId: string): UseOrderbookResult {
  const [orderbook, setOrderbook] = useState<OrderbookData | null>(null);
  const [trades, setTrades] = useState<Trade[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const requestIdRef = useRef(0);
  const controllerRef = useRef<AbortController | null>(null);
  const backoffRef = useRef(2000);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null); // #9: useRef for timer

  useEffect(() => {
    if (!marketId) return;
    const requestId = ++requestIdRef.current;
    backoffRef.current = 2000;

    const fetchData = async () => {
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;

      setLoading(true);
      setError(null);
      try {
        const [ob, tr] = await Promise.allSettled([
          fetchOrderbook(marketId),
          fetchRecentTrades(marketId, 100),
        ]);

        if (controller.signal.aborted || requestId !== requestIdRef.current) return;

        if (ob.status === "fulfilled") setOrderbook(ob.value);
        if (tr.status === "fulfilled") setTrades(tr.value.trades ?? []);

        if (ob.status === "rejected" && tr.status === "rejected") {
          setError("Failed to load orderbook data");
        }

        backoffRef.current = 2000;
      } catch (err) {
        if (controller.signal.aborted || requestId !== requestIdRef.current) return;
        setError(err instanceof Error ? err.message : "Failed to load orderbook");
        backoffRef.current = Math.min(backoffRef.current * 2, 30000);
      } finally {
        if (requestId === requestIdRef.current) setLoading(false);
      }
    };

    fetchData();

    const scheduleNext = () => {
      timerRef.current = setTimeout(() => {
        if (requestIdRef.current === requestId) {
          fetchData().then(() => {
            if (requestIdRef.current === requestId) {
              scheduleNext();
            }
          });
        }
      }, backoffRef.current);
    };

    scheduleNext();

    return () => {
      requestIdRef.current++;
      controllerRef.current?.abort();
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [marketId]);

  return { orderbook, trades, loading, error };
}
