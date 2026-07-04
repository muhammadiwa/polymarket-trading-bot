"use client";

import { AdminGuard } from "@/lib/auth/auth-guard";

export default function AdminPage() {
  return (
    <AdminGuard>
      <main className="mx-auto max-w-7xl px-4 py-8 space-y-8">
        <header className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-white">Admin Panel</h1>
        </header>

        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
            <h2 className="mb-4 text-lg font-semibold text-white">System Status</h2>
            <p className="text-gray-400">All systems operational</p>
          </div>

          <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
            <h2 className="mb-4 text-lg font-semibold text-white">User Management</h2>
            <p className="text-gray-400">Manage user accounts and roles</p>
          </div>

          <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
            <h2 className="mb-4 text-lg font-semibold text-white">Configuration</h2>
            <p className="text-gray-400">System configuration and settings</p>
          </div>
        </div>
      </main>
    </AdminGuard>
  );
}
