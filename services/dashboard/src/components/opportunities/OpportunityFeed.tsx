"use client";

import { useRef, useCallback, useMemo } from "react";
import { useOpportunityFeed, type OpportunityStatusFilter } from "@/hooks/useOpportunityFeed";
import type { Opportunity } from "@/types";

function statusBadgeColor(status: Opportunity["status"]): string {
  if (status === "detected") return "bg-[#00d4ff]/15 text-[#00d4ff]";
  if (status === "executed") return "bg-[#00ff88]/15 text-[#00ff88]";
  return "bg-gray-500/15 text-gray-400";
}

// NOTE: Virtual scrolling with @tanstack/react-virtual is deferred as a future
// enhancement. Current implementation uses standard DOM with max-height + overflow
// which is sufficient for typical feed sizes (< 1000 rows).

function formatScore(score: string): string {
  const num = Number(score);
  if (isNaN(num) || !isFinite(num)) return score;
  return num.toFixed(4);
}

function formatSpread(spread: string): string {
  const num = Number(spread);
  if (isNaN(num) || !isFinite(num)) return spread;
  return `$${num.toFixed(2)}`;
}

function formatTime(timestamp: string): string {
  try {
    return new Date(timestamp).toLocaleTimeString("en-US", {
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch {
    return timestamp;
  }
}

function OpportunityRow({ opp }: { opp: Opportunity }) {
  return (
    <div className="flex items-center gap-4 px-4 py-3 border-b border-white/5 hover:bg-white/[0.02] transition-colors">
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-white truncate" title={opp.market}>
          {opp.market}
        </p>
        {opp.filterReason && opp.status === "filtered" && (
          <p className="text-xs text-gray-500 truncate mt-0.5">{opp.filterReason}</p>
        )}
      </div>

      <div className="text-right shrink-0 w-20">
        <p className="text-xs text-gray-400">Score</p>
        <p className="text-sm font-mono text-white">{formatScore(opp.score)}</p>
      </div>

      <div className="text-right shrink-0 w-20">
        <p className="text-xs text-gray-400">Spread</p>
        <p className="text-sm font-mono text-white">{formatSpread(opp.spread)}</p>
      </div>

      <div className="text-right shrink-0 w-16">
        <p className="text-xs text-gray-400">Time</p>
        <p className="text-xs font-mono text-gray-300">{formatTime(opp.timestamp)}</p>
      </div>

      <div className="shrink-0">
        <span className={`inline-block text-xs px-2 py-0.5 rounded-full font-medium ${statusBadgeColor(opp.status)}`}>
          {opp.status.charAt(0).toUpperCase() + opp.status.slice(1)}
        </span>
      </div>

      {opp.executionLatencyMs !== null && opp.executionLatencyMs !== undefined && (
        <div className="text-right shrink-0 w-16">
          <p className="text-xs text-gray-400">Latency</p>
          <p className="text-xs font-mono text-gray-300">{opp.executionLatencyMs}ms</p>
        </div>
      )}
    </div>
  );
}

function FilterButtons({
  filter,
  setFilter,
}: {
  filter: OpportunityStatusFilter;
  setFilter: (f: OpportunityStatusFilter) => void;
}) {
  const filters: { value: OpportunityStatusFilter; label: string }[] = [
    { value: "all", label: "All" },
    { value: "detected", label: "Detected" },
    { value: "executed", label: "Executed" },
    { value: "filtered", label: "Filtered" },
  ];

  return (
    <div className="flex gap-1" role="tablist" aria-label="Filter opportunities">
      {filters.map(({ value, label }) => (
        <button
          key={value}
          onClick={() => setFilter(value)}
          className={`text-xs px-3 py-1 rounded-lg transition-colors ${
            filter === value
              ? "bg-white/10 text-white"
              : "text-gray-400 hover:text-white hover:bg-white/5"
          }`}
          role="tab"
          aria-selected={filter === value}
        >
          {label}
        </button>
      ))}
    </div>
  );
}

export function OpportunityFeed() {
  const { opportunities, loading, error, loadMore, hasMore, filter, setFilter } = useOpportunityFeed();
  const scrollRef = useRef<HTMLDivElement>(null);

  // #3: Client-side filter for display (WS stores all, filter here)
  const filteredOpportunities = useMemo(() => {
    if (filter === "all") return opportunities;
    return opportunities.filter((o) => o.status === filter);
  }, [opportunities, filter]);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el || !hasMore || loading) return;
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
      loadMore();
    }
  }, [hasMore, loading, loadMore]);

  return (
    <section className="space-y-4" aria-label="Opportunity Feed">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Opportunity Feed</h2>
        <FilterButtons filter={filter} setFilter={setFilter} />
      </div>

      <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md overflow-hidden">
        <div className="px-4 py-2 border-b border-white/10 flex items-center gap-4 text-xs text-gray-400">
          <div className="flex-1">Market</div>
          <div className="w-20 text-right">Score</div>
          <div className="w-20 text-right">Spread</div>
          <div className="w-16 text-right">Time</div>
          <div className="w-20">Status</div>
          <div className="w-16" />
        </div>

        {error && (
          <div className="px-4 py-3 text-[#ff4757] text-sm" role="alert">
            Failed to load opportunities. Please try again.
          </div>
        )}

        <div
          ref={scrollRef}
          onScroll={handleScroll}
          className="max-h-[500px] overflow-y-auto"
          role="list"
          aria-label="Opportunity list"
        >
          {loading && filteredOpportunities.length === 0 && (
            <div className="px-4 py-8 text-center text-gray-400 text-sm" aria-busy="true">
              Loading opportunities...
            </div>
          )}

          {!loading && filteredOpportunities.length === 0 && (
            <div className="px-4 py-8 text-center text-gray-500 text-sm">
              No opportunities found
            </div>
          )}

          {filteredOpportunities.map((opp) => (
            <OpportunityRow key={opp.id} opp={opp} />
          ))}

          {hasMore && !loading && (
            <button
              onClick={loadMore}
              className="w-full px-4 py-3 text-center text-xs text-gray-400 hover:text-white hover:bg-white/5 transition-colors"
            >
              Load more
            </button>
          )}

          {loading && filteredOpportunities.length > 0 && (
            <div className="px-4 py-3 text-center text-gray-400 text-xs" aria-busy="true">
              Loading more...
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
