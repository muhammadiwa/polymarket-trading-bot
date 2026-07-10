"use client";

import Link from "next/link";

export default function AdminPage() {
  return (
    <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
      <Link
        href="/admin/config"
        className="rounded-lg border border-gray-800 bg-gray-900 p-6 transition-colors hover:border-gray-600"
      >
        <h2 className="mb-2 text-lg font-semibold text-white">
          System Configuration
        </h2>
        <p className="text-gray-400">
          Manage API keys, risk defaults, and notification settings
        </p>
      </Link>

      <Link
        href="/admin/health"
        className="rounded-lg border border-gray-800 bg-gray-900 p-6 transition-colors hover:border-gray-600"
      >
        <h2 className="mb-2 text-lg font-semibold text-white">
          System Health
        </h2>
        <p className="text-gray-400">
          Monitor CPU, memory, disk, network, and service status
        </p>
      </Link>

      <Link
        href="/admin/logs"
        className="rounded-lg border border-gray-800 bg-gray-900 p-6 transition-colors hover:border-gray-600"
      >
        <h2 className="mb-2 text-lg font-semibold text-white">
          Log Viewer
        </h2>
        <p className="text-gray-400">
          View and filter system logs with full-text search
        </p>
      </Link>

      <Link
        href="/admin/database"
        className="rounded-lg border border-gray-800 bg-gray-900 p-6 transition-colors hover:border-gray-600"
      >
        <h2 className="mb-2 text-lg font-semibold text-white">
          Database Management
        </h2>
        <p className="text-gray-400">
          Backup, restore, and cleanup database
        </p>
      </Link>

      <Link
        href="/admin/backtest"
        className="rounded-lg border border-gray-800 bg-gray-900 p-6 transition-colors hover:border-gray-600"
      >
        <h2 className="mb-2 text-lg font-semibold text-white">
          Backtesting
        </h2>
        <p className="text-gray-400">
          Test strategies with historical data and optimize parameters
        </p>
      </Link>

      <div className="rounded-lg border border-gray-800 bg-gray-900 p-6 opacity-50">
        <h2 className="mb-2 text-lg font-semibold text-white">
          User Management
        </h2>
        <p className="text-gray-400">Coming soon - Manage user accounts and roles</p>
      </div>
    </div>
  );
}
