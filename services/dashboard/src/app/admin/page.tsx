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

      <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
        <h2 className="mb-2 text-lg font-semibold text-white">
          User Management
        </h2>
        <p className="text-gray-400">Manage user accounts and roles</p>
      </div>
    </div>
  );
}
