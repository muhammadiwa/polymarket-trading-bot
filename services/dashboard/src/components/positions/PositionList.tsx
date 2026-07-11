"use client";

import { useMemo, useState } from "react";
import { Column, Table } from "@/components/ui/Table";
import { usePositions } from "@/hooks/usePositions";
import type { Position } from "@/types";
import Decimal from "decimal.js";

function formatPrice(value: string): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN()) return "0.0000";
    return num.toFixed(4);
  } catch {
    return "0.0000";
  }
}

function formatQuantity(value: string): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN()) return "0.00";
    return num.toFixed(2);
  } catch {
    return "0.00";
  }
}

function formatPnL(value: string): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN()) return "$0.00";
    const sign = num.isNeg() ? "-" : "+";
    return `${sign}$${num.abs().toFixed(2)}`;
  } catch {
    return "$0.00";
  }
}

function pnlColor(value: string): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN() || num.isZero()) return "text-gray-400";
    return num.isPos() ? "text-[#00ff88]" : "text-[#ff4757]";
  } catch {
    return "text-gray-400";
  }
}

const COLUMNS: Column<Position>[] = [
  {
    key: "market",
    header: "Market",
    sortable: true,
    render: (row) => <span className="text-white">{row.market}</span>,
    sortValue: (row) => row.market,
  },
  {
    key: "side",
    header: "Side",
    sortable: true,
    render: (row) => (
      <span
        className={`px-2 py-0.5 rounded text-xs font-medium ${
          row.side === "YES"
            ? "bg-[#00ff88]/10 text-[#00ff88]"
            : "bg-[#ff4757]/10 text-[#ff4757]"
        }`}
      >
        {row.side}
      </span>
    ),
    sortValue: (row) => row.side,
  },
  {
    key: "entryPrice",
    header: "Entry",
    sortable: true,
    render: (row) => <span className="text-gray-300">{formatPrice(row.entryPrice)}</span>,
    sortValue: (row) => {
      try { return row.entryPrice; } catch { return "0"; }
    },
  },
  {
    key: "currentPrice",
    header: "Current",
    sortable: true,
    render: (row) => <span className="text-white">{formatPrice(row.currentPrice)}</span>,
    sortValue: (row) => {
      try { return row.currentPrice; } catch { return "0"; }
    },
  },
  {
    key: "quantity",
    header: "Qty",
    sortable: true,
    render: (row) => <span className="text-gray-300">{formatQuantity(row.quantity)}</span>,
    sortValue: (row) => {
      try { return row.quantity; } catch { return "0"; }
    },
  },
  {
    key: "unrealizedPnL",
    header: "PnL",
    sortable: true,
    render: (row) => (
      <span className={pnlColor(row.unrealizedPnL)}>{formatPnL(row.unrealizedPnL)}</span>
    ),
    sortValue: (row) => {
      try { return row.unrealizedPnL; } catch { return "0"; }
    },
  },
];

export function PositionList() {
  const { data, loading, error } = usePositions();
  const [sortKey, setSortKey] = useState<string>("market");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");

  const handleSort = (key: string) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const sorted = useMemo(() => {
    const col = COLUMNS.find((c) => c.key === sortKey);
    if (!col?.sortValue) return data;
    return [...data].sort((a, b) => {
      const va = col.sortValue!(a);
      const vb = col.sortValue!(b);
      if (typeof va === "number" && typeof vb === "number") {
        if (isNaN(va)) return 1;
        if (isNaN(vb)) return -1;
        return sortDir === "asc" ? va - vb : vb - va;
      }
      // #19: Decimal string comparison
      try {
        const da = new Decimal(va as string);
        const db = new Decimal(vb as string);
        const cmp = da.comparedTo(db);
        return sortDir === "asc" ? cmp : -cmp;
      } catch {
        return sortDir === "asc"
          ? String(va).localeCompare(String(vb))
          : String(vb).localeCompare(String(va));
      }
    });
  }, [data, sortKey, sortDir]);

  if (loading) {
    return (
      <div className="space-y-2" aria-busy="true" aria-label="Loading positions">
        <h2 className="text-lg font-semibold text-white">Positions</h2>
        <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-8 animate-pulse">
          <div className="h-4 w-full bg-white/10 rounded mb-3" />
          <div className="h-4 w-3/4 bg-white/10 rounded mb-3" />
          <div className="h-4 w-1/2 bg-white/10 rounded" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 backdrop-blur-md p-5 text-[#ff4757]" role="alert">
        Failed to load positions. Please try again.
      </div>
    );
  }

  return (
    <section className="space-y-4" aria-label="Positions">
      <h2 className="text-lg font-semibold text-white">Positions</h2>
      <Table
        columns={COLUMNS}
        data={sorted}
        sortKey={sortKey}
        sortDir={sortDir}
        onSort={handleSort}
        emptyMessage="No active positions"
      />
    </section>
  );
}
