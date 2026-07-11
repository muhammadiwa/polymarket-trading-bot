"use client";

import { useCallback, useEffect, useState } from "react";
import type { SystemConfig, ConfigAuditLog } from "@/types";
import { fetchAdminConfigs, updateAdminConfig, fetchConfigAuditLogs } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { AdminGuard } from "@/lib/auth/auth-guard";

const CATEGORIES = [
  { value: "", label: "All Categories" },
  { value: "api_keys", label: "API Keys" },
  { value: "risk_defaults", label: "Risk Defaults" },
  { value: "notification_settings", label: "Notification Settings" },
];

export default function ConfigPage() {
  const [configs, setConfigs] = useState<SystemConfig[]>([]);
  const [selectedCategory, setSelectedCategory] = useState("");
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [editReason, setEditReason] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [auditLogs, setAuditLogs] = useState<ConfigAuditLog[]>([]);
  const [showAudit, setShowAudit] = useState(false);

  const loadConfigs = useCallback(async () => {
    try {
      const data = await fetchAdminConfigs(selectedCategory || undefined);
      setConfigs(data.configs);
    } catch (err) {
      setError("Failed to load configurations");
    }
  }, [selectedCategory]);

  useEffect(() => {
    loadConfigs();
  }, [loadConfigs]);

  const loadAuditLogs = async (key?: string) => {
    try {
      const data = await fetchConfigAuditLogs(key, 20);
      setAuditLogs(data.logs);
      setShowAudit(true);
    } catch (err) {
      setError("Failed to load audit logs");
    }
  };

  const handleEdit = (config: SystemConfig) => {
    setEditingKey(config.configKey);
    // Don't load sensitive values into state - show empty field instead
    setEditValue(
      config.isSensitive
        ? ""
        : typeof config.configValue === "object"
          ? JSON.stringify(config.configValue)
          : String(config.configValue)
    );
    setEditReason("");
    setError(null);
    setSuccess(null);
  };

  const handleCancel = () => {
    setEditingKey(null);
    setEditValue("");
    setEditReason("");
  };

  const handleSave = async (config: SystemConfig) => {
    setSaving(true);
    setError(null);
    setSuccess(null);

    try {
      let parsedValue: any = editValue;
      if (config.category === "risk_defaults") {
        parsedValue = parseFloat(editValue);
        if (isNaN(parsedValue)) {
          setError("Value must be a number");
          setSaving(false);
          return;
        }
      } else if (
        config.category === "notification_settings" &&
        ["critical_bypass_throttle", "enable_telegram", "enable_email"].includes(
          config.configKey
        )
      ) {
        parsedValue = editValue.toLowerCase() === "true";
      }

      await updateAdminConfig(config.configKey, {
        configValue: parsedValue,
        reason: editReason || undefined,
        expectedUpdatedAt: config.updatedAt,
      });

      setSuccess(`Configuration '${config.configKey}' updated successfully`);
      setEditingKey(null);
      setEditValue("");
      setEditReason("");
      loadConfigs();
    } catch (err: any) {
      if (err.message?.includes("409")) {
        setError(
          "Configuration was modified by another user. Please refresh and try again."
        );
      } else {
        setError(err.message || "Failed to update configuration");
      }
    } finally {
      setSaving(false);
    }
  };

  const maskValue = (value: string) => {
    if (!value || value.length < 8) return "****";
    return `${value.slice(0, 3)}...${value.slice(-4)}`;
  };

  const formatValue = (config: SystemConfig) => {
    if (config.isSensitive && typeof config.configValue === "string") {
      return maskValue(config.configValue);
    }
    if (typeof config.configValue === "object") {
      return JSON.stringify(config.configValue);
    }
    return String(config.configValue);
  };

  return (
    <AppShell>
    <AdminGuard>
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">
          System Configuration
        </h2>
        <button
          onClick={() => loadAuditLogs()}
          className="rounded-md bg-gray-800 px-4 py-2 text-sm text-gray-300 hover:bg-gray-700"
        >
          View Audit Log
        </button>
      </div>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">
          {error}
        </div>
      )}

      {success && (
        <div className="rounded-md bg-green-900/50 p-4 text-green-200">
          {success}
        </div>
      )}

      <div className="flex gap-2">
        {CATEGORIES.map((cat) => (
          <button
            key={cat.value}
            onClick={() => setSelectedCategory(cat.value)}
            className={`rounded-md px-3 py-1.5 text-sm ${
              selectedCategory === cat.value
                ? "bg-blue-600 text-white"
                : "bg-gray-800 text-gray-400 hover:bg-gray-700"
            }`}
          >
            {cat.label}
          </button>
        ))}
      </div>

      <div className="space-y-4">
        {configs.map((config) => (
          <div
            key={config.configKey}
            className="rounded-lg border border-gray-800 bg-gray-900 p-4"
          >
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-medium text-white">{config.configKey}</h3>
                {config.description && (
                  <p className="mt-1 text-sm text-gray-400">
                    {config.description}
                  </p>
                )}
                <div className="mt-2 flex gap-2">
                  <span className="rounded bg-gray-800 px-2 py-0.5 text-xs text-gray-400">
                    {config.category}
                  </span>
                  {config.isSensitive && (
                    <span className="rounded bg-yellow-900/50 px-2 py-0.5 text-xs text-yellow-400">
                      Sensitive
                    </span>
                  )}
                </div>
              </div>

              {editingKey === config.configKey ? (
                <div className="flex gap-2">
                  <button
                    onClick={handleCancel}
                    className="rounded-md bg-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-700"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => handleSave(config)}
                    disabled={saving}
                    className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
                  >
                    {saving ? "Saving..." : "Save"}
                  </button>
                </div>
              ) : (
                <div className="flex gap-2">
                  <button
                    onClick={() => loadAuditLogs(config.configKey)}
                    className="rounded-md bg-gray-800 px-3 py-1.5 text-sm text-gray-400 hover:bg-gray-700"
                  >
                    History
                  </button>
                  <button
                    onClick={() => handleEdit(config)}
                    className="rounded-md bg-gray-800 px-3 py-1.5 text-sm text-white hover:bg-gray-700"
                  >
                    Edit
                  </button>
                </div>
              )}
            </div>

            {editingKey === config.configKey ? (
              <div className="mt-4 space-y-3">
                <div>
                  <label className="block text-sm text-gray-400">Value</label>
                  <input
                    type={config.isSensitive ? "password" : "text"}
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    placeholder={config.isSensitive ? "Enter new value to change" : ""}
                    className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-blue-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm text-gray-400">
                    Reason (optional)
                  </label>
                  <input
                    type="text"
                    value={editReason}
                    onChange={(e) => setEditReason(e.target.value)}
                    placeholder="Why are you making this change?"
                    className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-blue-500 focus:outline-none"
                  />
                </div>
              </div>
            ) : (
              <div className="mt-3 rounded-md bg-gray-800 px-3 py-2 font-mono text-sm text-gray-300">
                {formatValue(config)}
              </div>
            )}
          </div>
        ))}
      </div>

      {showAudit && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="max-h-[80vh] w-full max-w-3xl overflow-hidden rounded-lg bg-gray-900">
            <div className="flex items-center justify-between border-b border-gray-800 p-4">
              <h3 className="text-lg font-semibold text-white">
                Configuration Audit Log
              </h3>
              <button
                onClick={() => setShowAudit(false)}
                className="text-gray-400 hover:text-white"
              >
                ✕
              </button>
            </div>
            <div className="max-h-[60vh] overflow-y-auto p-4">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-gray-400">
                    <th className="pb-2">Time</th>
                    <th className="pb-2">Key</th>
                    <th className="pb-2">Old Value</th>
                    <th className="pb-2">New Value</th>
                    <th className="pb-2">Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {auditLogs.map((log) => (
                    <tr key={log.id} className="border-t border-gray-800">
                      <td className="py-2 text-gray-400">
                        {new Date(log.changedAt).toLocaleString()}
                      </td>
                      <td className="py-2 text-white">{log.configKey}</td>
                      <td className="py-2 font-mono text-gray-400">
                        {JSON.stringify(log.oldValue)}
                      </td>
                      <td className="py-2 font-mono text-gray-300">
                        {JSON.stringify(log.newValue)}
                      </td>
                      <td className="py-2 text-gray-400">
                        {log.reason || "-"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {auditLogs.length === 0 && (
                <p className="py-4 text-center text-gray-400">
                  No audit logs found
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
    </AdminGuard>
    </AppShell>
  );
}
