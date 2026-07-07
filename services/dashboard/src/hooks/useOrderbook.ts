"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { fetchOrderbook, fetchRecentTrades } from "@/lib/api";

interface OrderbookLevel {
  price: string;
  size: string;
  cumulative: string;
}

interface OrderbookData {
  market_id: string;
  bids: OrderbookLevel[];
  asks: OrderbookLevel[];
  spread: string;
  last_update: string;
}

interface Trade {
  price: string;
  size: string;
  side: string;
  timestamp: string;
}

interface UseOrderbookResult {
  orderbook: OrderbookData | null;
  trades: Trade[];
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

  useEffect(() => {
    if (!marketId) return;
    const requestId = ++requestIdRef.current;
    backoffRef.current = 2000; // Reset backoff on new market

    const fetchData = async () => {
      // Cancel previous in-flight request
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;

      setLoading(true);
      setError(null);
      try {
        const [ob, tr] = await Promise.allSettled([
          fetchOrderbook(marketId),
          fetchRecentTrades(marketId, 100), // #5: Explicitly pass limit=100
        ]);

        if (controller.signal.aborted || requestId !== requestIdRef.current) return;

        if (ob.status === "fulfilled") setOrderbook(ob.value);
        if (tr.status === "fulfilled") setTrades(tr.value.trades ?? []);

        if (ob.status === "rejected" && tr.status === "rejected") {
          setError("Failed to load orderbook data");
        }

        backoffRef.current = 2000; // Reset on success
      } catch (err) {
        if (controller.signal.aborted || requestId !== requestIdRef.current) return;
        setError(err instanceof Error ? err.message : "Failed to load orderbook");
        backoffRef.current = Math.min(backoffRef.current * 2, 30000); // Exponential backoff
      } finally {
        if (requestId === requestIdRef.current) setLoading(false);
      }
    };

    fetchData();

    // Poll with backoff-aware interval
    const scheduleNext = () => {
      return setTimeout(() => {
        if (requestIdRef.current === requestId) {
          fetchData().then(() => {
            if (requestIdRef.current === requestId) {
              intervalRef.current = scheduleNext();
            }
          });
        }
      }, backoffRef.current);
    };

    let intervalRef = { current: scheduleNext() };

    return () => {
      requestIdRef.current++;
      controllerRef.current?.abort();
      clearTimeout(intervalRef.current);
    };
  }, [marketId]);

  return { orderbook, trades, loading, error };
}
