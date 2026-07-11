"use client";

import { AuthGuard } from "@/lib/auth/auth-guard";
import { WSProvider } from "@/lib/ws-context";
import { Sidebar } from "@/components/layout/Sidebar";
import { Header } from "@/components/layout/Header";

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard>
      <WSProvider>
        <div className="app-shell">
          <Sidebar />
          <div className="app-main">
            <Header />
            <main className="app-content">
              {children}
            </main>
          </div>
        </div>
      </WSProvider>
    </AuthGuard>
  );
}
