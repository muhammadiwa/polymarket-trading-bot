"use client";

import { useCallback, useEffect, useState } from "react";
import type { BackupInfo, DatabaseStats } from "@/types";
import {
  createBackup,
  fetchBackups,
  getRestoreConfirmToken,
  restoreBackup,
  cleanupDatabase,
  fetchDatabaseStats,
} from "@/lib/api";
import { AdminGuard } from "@/lib/auth/auth-guard";

const ELIGIBLE_TABLES = [
  { name: "system_logs", retention: 90, description: "Application logs" },
  { name: "trades", retention: 2555, description: "Trade history (7 years)" },
  { name: "opportunities", retention: 1095, description: "Detected opportunities (3 years)" },
  { name: "risk_events", retention: 365, description: "Risk decision logs (1 year)" },
  { name: "config_audit_log", retention: 365, description: "Config change history (1 year)" },
  { name: "notifications", retention: 90, description: "Notification history" },
];

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatDuration(ms: number | null): string {
  if (!ms) return "-";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

export default function DatabasePage() {
  const [stats, setStats] = useState<DatabaseStats | null>(null);
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Backup state
  const [backing, setBacking] = useState(false);

  // Restore state
  const [restoringId, setRestoringId] = useState<string | null>(null);
  const [confirmRestoreId, setConfirmRestoreId] = useState<string | null>(null);
  const [confirmationToken, setConfirmationToken] = useState<string | null>(null);

  // Cleanup state
  const [cleanupDays, setCleanupDays] = useState(90);
  const [selectedTables, setSelectedTables] = useState<string[]>([]);
  const [cleaning, setCleaning] = useState(false);

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [statsData, backupsData] = await Promise.all([
        fetchDatabaseStats(),
        fetchBackups(),
      ]);
      setStats(statsData);
      setBackups(backupsData.backups);
    } catch (err) {
      setError("Failed to load database info");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleBackup = async () => {
    setBacking(true);
    setError(null);
    setSuccess(null);
    try {
      await createBackup();
      setSuccess("Backup created successfully");
      loadData();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Backup failed");
    } finally {
      setBacking(false);
    }
  };

  const handleRestoreClick = async (backupId: string) => {
    try {
      setError(null);
      const { confirmationToken: token } = await getRestoreConfirmToken(backupId);
      setConfirmRestoreId(backupId);
      setConfirmationToken(token);
    } catch (err: any) {
      setError(err.message || "Failed to generate confirmation token");
    }
  };

  const handleRestoreConfirm = async () => {
    if (!confirmRestoreId || !confirmationToken) return;

    setRestoringId(confirmRestoreId);
    setError(null);
    setSuccess(null);
    try {
      await restoreBackup(confirmRestoreId, confirmationToken);
      setSuccess("Database restored successfully");
      setConfirmRestoreId(null);
      setConfirmationToken(null);
      loadData();
    } catch (err: any) {
      setError(err.message || "Restore failed");
    } finally {
      setRestoringId(null);
    }
  };

  const handleCleanup = async () => {
    setCleaning(true);
    setError(null);
    setSuccess(null);
    try {
      const result = await cleanupDatabase(
        cleanupDays,
        selectedTables.length > 0 ? selectedTables : undefined
      );
      const totalDeleted = Object.values(result.deletedRows).reduce((a, b) => a + b, 0);
      setSuccess(`Cleanup complete: ${totalDeleted} rows deleted, ${formatBytes(result.freedBytes)} freed`);
      loadData();
    } catch (err: any) {
      setError(err.message || "Cleanup failed");
    } finally {
      setCleaning(false);
    }
  };

  const toggleTable = (table: string) => {
    setSelectedTables((prev) =>
      prev.includes(table) ? prev.filter((t) => t !== table) : [...prev, table]
    );
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-400">Loading database info...</div>
      </div>
    );
  }

  return (
    <AdminGuard>
    <div className="space-y-8">
      <h2 className="text-xl font-semibold text-white">Database Management</h2>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">{error}</div>
      )}
      {success && (
        <div className="rounded-md bg-green-900/50 p-4 text-green-200">{success}</div>
      )}

      {/* Database Stats */}
      {stats && (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
            <h3 className="text-sm text-gray-400">Total Size</h3>
            <p className="mt-1 text-2xl font-bold text-white">
              {formatBytes(stats.totalSizeBytes)}
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
            <h3 className="text-sm text-gray-400">Log Entries</h3>
            <p className="mt-1 text-2xl font-bold text-white">
              {stats.totalLogEntries.toLocaleString()}
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
            <h3 className="text-sm text-gray-400">Trades</h3>
            <p className="mt-1 text-2xl font-bold text-white">
              {stats.totalTrades.toLocaleString()}
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
            <h3 className="text-sm text-gray-400">Positions</h3>
            <p className="mt-1 text-2xl font-bold text-white">
              {stats.totalPositions.toLocaleString()}
            </p>
          </div>
        </div>
      )}

      {/* Table Sizes */}
      {stats && (
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <h3 className="mb-4 text-lg font-semibold text-white">Table Sizes</h3>
          <div className="space-y-2">
            {Object.entries(stats.tableSizes).map(([table, size]) => (
              <div key={table} className="flex items-center justify-between">
                <span className="text-gray-400">{table}</span>
                <span className="text-white">{formatBytes(size)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Backup Section */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-white">Backups</h3>
          <button
            onClick={handleBackup}
            disabled={backing}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {backing ? "Creating..." : "Create Backup"}
          </button>
        </div>

        <div className="mt-4 space-y-2">
          {backups.length === 0 ? (
            <p className="text-gray-400">No backups found</p>
          ) : (
            backups.map((backup) => (
              <div
                key={backup.id}
                className="flex items-center justify-between rounded-md border border-gray-800 p-3"
              >
                <div>
                  <p className="text-white">{backup.filename}</p>
                  <p className="text-sm text-gray-400">
                    {formatBytes(backup.sizeBytes)} •{" "}
                    {new Date(backup.createdAt).toLocaleString()} •{" "}
                    {formatDuration(backup.durationMs)}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={`rounded px-2 py-0.5 text-xs ${
                      backup.status === "completed"
                        ? "bg-green-900/50 text-green-400"
                        : backup.status === "failed"
                          ? "bg-red-900/50 text-red-400"
                          : "bg-yellow-900/50 text-yellow-400"
                    }`}
                  >
                    {backup.status}
                  </span>
                  {backup.status === "completed" && (
                    <button
                      onClick={() => handleRestoreClick(backup.id)}
                      disabled={restoringId === backup.id}
                      className="rounded-md bg-gray-800 px-3 py-1 text-sm text-gray-300 hover:bg-gray-700 disabled:opacity-50"
                    >
                      {restoringId === backup.id ? "Restoring..." : "Restore"}
                    </button>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Restore Confirmation Dialog */}
      {confirmRestoreId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg bg-gray-900 p-6">
            <h3 className="text-lg font-semibold text-red-400">⚠️ Confirm Restore</h3>
            <p className="mt-4 text-gray-300">
              This will <strong>overwrite the current database</strong> with the
              selected backup. This action cannot be undone.
            </p>
            <p className="mt-2 text-sm text-gray-400">
              All current data not in the backup will be lost.
            </p>
            <div className="mt-6 flex justify-end gap-4">
              <button
                onClick={() => {
                  setConfirmRestoreId(null);
                  setConfirmationToken(null);
                }}
                className="rounded-md bg-gray-800 px-4 py-2 text-sm text-gray-300 hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={handleRestoreConfirm}
                className="rounded-md bg-red-600 px-4 py-2 text-sm text-white hover:bg-red-700"
              >
                Restore Database
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Cleanup Section */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
        <h3 className="text-lg font-semibold text-white">Cleanup Old Data</h3>
        <p className="mt-2 text-sm text-gray-400">
          Delete data older than the specified retention period.
        </p>

        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm text-gray-400">Retention (days)</label>
            <input
              type="number"
              min={1}
              max={365}
              value={cleanupDays}
              onChange={(e) => setCleanupDays(parseInt(e.target.value) || 90)}
              className="mt-1 w-32 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400">
              Tables (leave empty for all)
            </label>
            <div className="mt-2 flex flex-wrap gap-2">
              {ELIGIBLE_TABLES.map((table) => (
                <button
                  key={table.name}
                  onClick={() => toggleTable(table.name)}
                  className={`rounded-md px-3 py-1.5 text-sm ${
                    selectedTables.includes(table.name)
                      ? "bg-blue-600 text-white"
                      : "bg-gray-800 text-gray-400 hover:bg-gray-700"
                  }`}
                >
                  {table.name}
                </button>
              ))}
            </div>
          </div>

          <button
            onClick={handleCleanup}
            disabled={cleaning}
            className="rounded-md bg-yellow-600 px-4 py-2 text-sm text-white hover:bg-yellow-700 disabled:opacity-50"
          >
            {cleaning ? "Cleaning..." : "Run Cleanup"}
          </button>
        </div>
      </div>
    </div>
    </AdminGuard>
  );
}
