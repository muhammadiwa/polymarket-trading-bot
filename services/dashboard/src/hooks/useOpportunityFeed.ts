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
  const loadingRef = useRef(false);
  const hasMoreRef = useRef(true);

  const loadInitial = useCallback(async () => {
    setLoading(true);
    loadingRef.current = true;
    try {
      const resp = await fetchOpportunities(undefined, 50, filter === "all" ? undefined : filter);
      // #4: Filter WS-only items by current filter
      setOpportunities((prev) => {
        const existingIds = new Set(resp.opportunities.map((o) => o.id));
        const wsOnly = prev.filter((o) => !existingIds.has(o.id) && (filter === "all" || o.status === filter));
        return [...resp.opportunities, ...wsOnly];
      });
      setNextCursor(resp.next_cursor);
      setHasMore(resp.next_cursor !== null);
      hasMoreRef.current = resp.next_cursor !== null;
      cursorRef.current = resp.next_cursor;
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load opportunities");
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  }, [filter]);

  // #5: loadMore with loading guard
  const loadMore = useCallback(async () => {
    if (!cursorRef.current || loadingRef.current) return;
    loadingRef.current = true;
    try {
      const resp = await fetchOpportunities(cursorRef.current, 50, filter === "all" ? undefined : filter);
      setOpportunities((prev) => [...prev, ...resp.opportunities]);
      setNextCursor(resp.next_cursor);
      setHasMore(resp.next_cursor !== null);
      hasMoreRef.current = resp.next_cursor !== null;
      cursorRef.current = resp.next_cursor;
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load more opportunities");
    } finally {
      loadingRef.current = false;
    }
  }, [filter]);

  useEffect(() => {
    loadInitial();
  }, [loadInitial]);

  useEffect(() => {
    const unsubscribe = onOpportunity((opp) => {
      // #3: Always store WS opportunities, filter at render time
      setOpportunities((prev) => {
        if (prev.some((o) => o.id === opp.id)) return prev;
        return [opp, ...prev];
      });
    });
    return unsubscribe;
  }, [onOpportunity]);

  return { opportunities, loading, error, loadMore, hasMore, filter, setFilter };
}
