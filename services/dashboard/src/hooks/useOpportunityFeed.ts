"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { fetchOpportunities } from "@/lib/api";
import { useWSContext } from "@/lib/ws-context";
import type { Opportunity } from "@/types";

interface UseOpportunityFeedResult {
  opportunities: Opportunity[];
  loading: boolean;
  error: string | null;
  loadMore: () => Promise<void>;
  hasMore: boolean;
  filter: OpportunityStatusFilter;
  setFilter: (f: OpportunityStatusFilter) => void;
}

export type OpportunityStatusFilter = "all" | "detected" | "executed" | "filtered";

export function useOpportunityFeed(): UseOpportunityFeedResult {
  const { onOpportunity } = useWSContext();
  const [opportunities, setOpportunities] = useState<Opportunity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [filter, setFilter] = useState<OpportunityStatusFilter>("all");
  const cursorRef = useRef<string | null>(null);

  const loadInitial = useCallback(async () => {
    setLoading(true);
    try {
      // #5: Pass status filter as query parameter to API instead of client-side filtering
      const resp = await fetchOpportunities(undefined, 50, filter === "all" ? undefined : filter);
      // #17: Deduplicate API results by id in case WS has already delivered some
      setOpportunities((prev) => {
        const existingIds = new Set(resp.opportunities.map((o) => o.id));
        const wsOnly = prev.filter((o) => !existingIds.has(o.id));
        return [...resp.opportunities, ...wsOnly];
      });
      setNextCursor(resp.next_cursor);
      setHasMore(resp.next_cursor !== null);
      cursorRef.current = resp.next_cursor;
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load opportunities");
    } finally {
      setLoading(false);
    }
  }, [filter]);

  const loadMore = useCallback(async () => {
    if (!cursorRef.current) return;
    try {
      const resp = await fetchOpportunities(cursorRef.current, 50, filter === "all" ? undefined : filter);
      setOpportunities((prev) => [...prev, ...resp.opportunities]);
      setNextCursor(resp.next_cursor);
      setHasMore(resp.next_cursor !== null);
      cursorRef.current = resp.next_cursor;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load more opportunities");
    }
  }, [filter]);

  useEffect(() => {
    loadInitial();
  }, [loadInitial]);

  useEffect(() => {
    const unsubscribe = onOpportunity((opp) => {
      if (filter === "all" || opp.status === filter) {
        // #19: Deduplicate by id on prepend
        setOpportunities((prev) => {
          if (prev.some((o) => o.id === opp.id)) return prev;
          return [opp, ...prev];
        });
      }
    });
    return unsubscribe;
  }, [onOpportunity, filter]);

  return { opportunities, loading, error, loadMore, hasMore, filter, setFilter };
}
