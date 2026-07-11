import type { Metadata } from "next";
import Decimal from "decimal.js";
import { WSProvider } from "@/lib/ws-context";
import { AuthProvider, AuthGuard } from "@/lib/auth/auth-guard";
import { ErrorBoundary } from "@/components/ui/ErrorBoundary";
import "./globals.css";

// #12: Set decimal precision once in shared entry point (not per-component)
Decimal.set({ precision: 20 });

export const metadata: Metadata = {
  title: "PQAP Dashboard",
  description: "Portfolio & Position Dashboard",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="min-h-screen antialiased">
        <ErrorBoundary>
          <AuthProvider>
            <AuthGuard>
              <WSProvider>{children}</WSProvider>
            </AuthGuard>
          </AuthProvider>
        </ErrorBoundary>
      </body>
    </html>
  );
}
