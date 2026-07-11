"use client";

import { AuthGuard } from "@/lib/auth/auth-guard";
import { OrderbookView } from "@/components/orderbook/OrderbookView";

export default function OrderbookPage() {
  return (
    <AuthGuard>
      <div className="p-6">
        <OrderbookView />
      </div>
    </AuthGuard>
  );
}
