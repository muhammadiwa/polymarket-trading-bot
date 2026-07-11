"use client";

import { useCallback, useEffect, useState } from "react";
import type { SystemLog, LogQueryParams } from "@/types";
import { fetchAdminLogs, fetchLogServices } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { AdminGuard } from "@/lib/auth/auth-guard";

const LOG_LEVELS = [
  { value: "", label: "All Levels" },
  { value: "debug", label: "Debug" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warning" },
  { value: "error", label: "Error" },
  { value: "fatal", label: "Fatal" },
];

const LEVEL_COLORS: Record<string, string> = {
  debug: "text-gray-400 bg-gray-800",
  info: "text-blue-400 bg-blue-900/50",
  warn: "text-yellow-400 bg-yellow-900/50",
  error: "text-red-400 bg-red-900/50",
  fatal: "text-red-300 bg-red-800/50",
};

export default function LogsPage() {
  const [logs, setLogs] = useState<SystemLog[]>([]);
  const [services, setServices] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  const [filters, setFilters] = useState<LogQueryParams>({
    limit: 100,
    offset: 0,
  });

  const [searchText, setSearchText] = useState("");
  const [autoRefresh, setAutoRefresh] = useState(false);

  const loadLogs = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchAdminLogs(filters);
      setLogs(data.logs);
      setTotal(data.total);
      setHasMore(data.hasMore);
    } catch (err) {
      setError("Failed to load logs");
    } finally {
      setLoading(false);
    }
  }, [filters]);

  const loadServices = async () => {
    try {
      const data = await fetchLogServices();
      setServices(data);
    } catch (err) {
      console.error("Failed to load services:", err);
    }
  };

  useEffect(() => {
    loadServices();
  }, []);

  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(loadLogs, 5000);
    return () => clearInterval(interval);
  }, [autoRefresh, loadLogs]);

  const handleSearch = () => {
    setFilters((prev) => ({ ...prev, search: searchText, offset: 0 }));
  };

  const handleFilterChange = (key: keyof LogQueryParams, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value || undefined, offset: 0 }));
  };

  const handleLoadMore = () => {
    setFilters((prev) => ({ ...prev, offset: (prev.offset || 0) + (prev.limit || 100) }));
  };

  const formatTimestamp = (ts: string) => {
    return new Date(ts).toLocaleString();
  };

  return (
    <AppShell>
    <AdminGuard>
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">System Logs</h2>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm text-gray-400">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="rounded border-gray-600 bg-gray-800"
            />
            Auto-refresh
          </label>
          <button
            onClick={loadLogs}
            className="rounded-md bg-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-700"
          >
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">{error}</div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap gap-4">
        <div>
          <label className="block text-sm text-gray-400">Level</label>
          <select
            value={filters.level || ""}
            onChange={(e) => handleFilterChange("level", e.target.value)}
            className="mt-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
          >
            {LOG_LEVELS.map((l) => (
              <option key={l.value} value={l.value}>
                {l.label}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm text-gray-400">Service</label>
          <select
            value={filters.service || ""}
            onChange={(e) => handleFilterChange("service", e.target.value)}
            className="mt-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
          >
            <option value="">All Services</option>
            {services.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm text-gray-400">Start Date</label>
          <input
            type="datetime-local"
            value={filters.startDate || ""}
            onChange={(e) => handleFilterChange("startDate", e.target.value)}
            className="mt-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400">End Date</label>
          <input
            type="datetime-local"
            value={filters.endDate || ""}
            onChange={(e) => handleFilterChange("endDate", e.target.value)}
            className="mt-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
          />
        </div>

        <div className="flex-1">
          <label className="block text-sm text-gray-400">Search</label>
          <div className="flex gap-2">
            <input
              type="text"
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              placeholder="Search messages..."
              className="mt-1 flex-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
            />
            <button
              onClick={handleSearch}
              className="mt-1 rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700"
            >
              Search
            </button>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="text-sm text-gray-400">
        Showing {logs.length} of {total} logs
      </div>

      {/* Log Table */}
      <div className="overflow-x-auto rounded-lg border border-gray-800">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800 bg-gray-900 text-left text-gray-400">
              <th className="px-4 py-3">Timestamp</th>
              <th className="px-4 py-3">Level</th>
              <th className="px-4 py-3">Service</th>
              <th className="px-4 py-3">Request ID</th>
              <th className="px-4 py-3">Message</th>
            </tr>
          </thead>
          <tbody>
            {loading && logs.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-400">
                  Loading...
                </td>
              </tr>
            ) : logs.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-400">
                  No logs found
                </td>
              </tr>
            ) : (
              logs.map((log) => (
                <tr key={log.id} className="border-b border-gray-800 hover:bg-gray-900/50">
                  <td className="whitespace-nowrap px-4 py-3 text-gray-400">
                    {formatTimestamp(log.timestamp)}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`rounded px-2 py-0.5 text-xs font-medium ${LEVEL_COLORS[log.level] || "text-gray-400 bg-gray-800"}`}
                    >
                      {log.level.toUpperCase()}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-white">{log.service}</td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-500">
                    {log.requestId || "-"}
                  </td>
                  <td className="max-w-md truncate px-4 py-3 text-gray-300">
                    {log.message}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Load More */}
      {hasMore && (
        <div className="flex justify-center">
          <button
            onClick={handleLoadMore}
            className="rounded-md bg-gray-800 px-6 py-2 text-sm text-gray-300 hover:bg-gray-700"
          >
            Load More
          </button>
        </div>
      )}
    </div>
    </AdminGuard>
    </AppShell>
  );
}
