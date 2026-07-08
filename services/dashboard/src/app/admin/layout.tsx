"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { AdminGuard } from "@/lib/auth/auth-guard";
import { ErrorBoundary } from "@/components/ui/ErrorBoundary";

const navItems = [
  { href: "/admin", label: "Overview" },
  { href: "/admin/config", label: "Configuration" },
  { href: "/admin/health", label: "System Health" },
];

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();

  return (
    <AdminGuard>
      <main className="mx-auto max-w-7xl px-4 py-8 space-y-8">
        <header className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link href="/" className="text-gray-400 hover:text-white">
              ← Dashboard
            </Link>
            <h1 className="text-2xl font-bold text-white">Admin Panel</h1>
          </div>
        </header>

        <nav className="flex gap-1 rounded-lg bg-gray-900 p-1">
          {navItems.map((item) => {
            const isActive =
              pathname === item.href ||
              (item.href !== "/admin" && pathname.startsWith(item.href));
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`rounded-md px-4 py-2 text-sm font-medium ${
                  isActive
                    ? "bg-gray-800 text-white"
                    : "text-gray-400 hover:bg-gray-800 hover:text-white"
                }`}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>

        <ErrorBoundary>{children}</ErrorBoundary>
      </main>
    </AdminGuard>
  );
}
