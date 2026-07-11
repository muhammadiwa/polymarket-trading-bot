"use client";

import { AppShell } from "@/components/layout/AppShell";
import { OrderbookView } from "@/components/orderbook/OrderbookView";

export default function OrderbookPage() {
  return (
    <AppShell>
      <div className="p-6">
        <OrderbookView />
      </div>
    </AppShell>
  );
}
