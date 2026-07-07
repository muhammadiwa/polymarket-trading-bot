"use client";

import { useEffect, useState, useRef } from "react";
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

  useEffect(() => {
    if (!marketId) return;
    const requestId = ++requestIdRef.current;

    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [ob, tr] = await Promise.all([
          fetchOrderbook(marketId),
          fetchRecentTrades(marketId),
        ]);
        if (requestId !== requestIdRef.current) return;
        setOrderbook(ob);
        setTrades(tr.trades ?? []);
      } catch (err) {
        if (requestId !== requestIdRef.current) return;
        setError(err instanceof Error ? err.message : "Failed to load orderbook");
      } finally {
        if (requestId === requestIdRef.current) setLoading(false);
      }
    };

    fetchData();

    // Poll every 2 seconds
    const interval = setInterval(fetchData, 2000);
    return () => {
      clearInterval(interval);
    };
  }, [marketId]);

  return { orderbook, trades, loading, error };
}
