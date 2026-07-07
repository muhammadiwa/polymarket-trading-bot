"use client";

import { useState, useCallback } from "react";
import { OrderbookTable } from "./OrderbookTable";
import { DepthChart } from "./DepthChart";
import { RecentTrades } from "./RecentTrades";
import { useOrderbook } from "@/hooks/useOrderbook";

interface Tab {
  marketId: string;
  label: string;
}

export function OrderbookView() {
  const [tabs, setTabs] = useState<Tab[]>([]);
  const [activeTab, setActiveTab] = useState<string | null>(null);
  const [marketInput, setMarketInput] = useState("");

  const addTab = useCallback(() => {
    if (!marketInput.trim()) return;
    if (tabs.length >= 5) return;
    if (tabs.some((t) => t.marketId === marketInput.trim())) return;

    const newTab = { marketId: marketInput.trim(), label: marketInput.trim().slice(0, 12) };
    setTabs((prev) => [...prev, newTab]);
    setActiveTab(newTab.marketId);
    setMarketInput("");
  }, [marketInput, tabs]);

  const removeTab = useCallback((marketId: string) => {
    setTabs((prev) => prev.filter((t) => t.marketId !== marketId));
    setActiveTab((prev) => (prev === marketId ? tabs[0]?.marketId ?? null : prev));
  }, [tabs]);

  return (
    <section className="space-y-4" aria-label="Orderbook Viewer">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Orderbook</h2>
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder="Market ID"
            value={marketInput}
            onChange={(e) => setMarketInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addTab()}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm w-48"
          />
          <button
            onClick={addTab}
            disabled={tabs.length >= 5 || !marketInput.trim()}
            className="px-4 py-2 rounded-lg bg-[#00d4ff] text-black font-medium text-sm hover:bg-[#00d4ff]/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Open
          </button>
        </div>
      </div>

      {/* Tabs */}
      {tabs.length > 0 && (
        <div className="flex gap-1 border-b border-white/10">
          {tabs.map((tab) => (
            <button
              key={tab.marketId}
              onClick={() => setActiveTab(tab.marketId)}
              className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${
                activeTab === tab.marketId
                  ? "bg-white/10 text-white border-b-2 border-[#00d4ff]"
                  : "text-gray-400 hover:text-white"
              }`}
            >
              {tab.label}
              <span
                onClick={(e) => { e.stopPropagation(); removeTab(tab.marketId); }}
                className="ml-2 text-gray-500 hover:text-[#ff4757] cursor-pointer"
              >
                ×
              </span>
            </button>
          ))}
        </div>
      )}

      {/* Active tab content */}
      {activeTab ? (
        <OrderbookTabContent marketId={activeTab} />
      ) : (
        <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-8 text-center text-gray-400">
          Enter a Market ID to view the orderbook
        </div>
      )}
    </section>
  );
}

function OrderbookTabContent({ marketId }: { marketId: string }) {
  const { orderbook, trades, loading, error } = useOrderbook(marketId);

  if (loading) {
    return <div className="animate-pulse bg-white/5 rounded-xl h-96" />;
  }

  if (error) {
    return (
      <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 p-5 text-[#ff4757]">
        {error}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <div className="lg:col-span-2 space-y-4">
        <OrderbookTable bids={orderbook?.bids ?? []} asks={orderbook?.asks ?? []} spread={orderbook?.spread ?? "0"} />
        <DepthChart bids={orderbook?.bids ?? []} asks={orderbook?.asks ?? []} />
      </div>
      <div>
        <RecentTrades trades={trades ?? []} />
      </div>
    </div>
  );
}
