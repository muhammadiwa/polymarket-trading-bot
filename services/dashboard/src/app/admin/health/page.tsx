"use client";

import { useCallback, useEffect, useState } from "react";
import type { AdminHealthStatus, HealthAlert, ServiceHealth } from "@/types";
import { fetchAdminHealth } from "@/lib/api";

export default function HealthPage() {
  const [health, setHealth] = useState<AdminHealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadHealth = useCallback(async () => {
    try {
      const data = await fetchAdminHealth();
      setHealth(data);
      setError(null);
    } catch (err) {
      setError("Failed to load health status");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadHealth();
    const interval = setInterval(loadHealth, 5000);
    return () => clearInterval(interval);
  }, [loadHealth]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case "up":
      case "healthy":
        return "text-green-400 bg-green-900/50";
      case "degraded":
        return "text-yellow-400 bg-yellow-900/50";
      case "down":
      case "unhealthy":
        return "text-red-400 bg-red-900/50";
      default:
        return "text-gray-400 bg-gray-800";
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case "warning":
        return "text-yellow-400 bg-yellow-900/50";
      case "critical":
        return "text-red-400 bg-red-900/50";
      default:
        return "text-gray-400 bg-gray-800";
    }
  };

  const services: (ServiceHealth | undefined)[] = health
    ? [
        health.scanner,
        health.arbEngine,
        health.executionEngine,
        health.riskManager,
        health.positionManager,
      ]
    : [];

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-400">Loading health status...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">System Health</h2>
        <div className="flex items-center gap-4">
          {health && (
            <span
              className={`rounded-md px-3 py-1 text-sm font-medium ${getStatusColor(health.overall)}`}
            >
              {health.overall.toUpperCase()}
            </span>
          )}
          <button
            onClick={loadHealth}
            className="rounded-md bg-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-700"
          >
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {services.map(
          (service) =>
            service && (
              <div
                key={service.name}
                className="rounded-lg border border-gray-800 bg-gray-900 p-4"
              >
                <div className="flex items-center justify-between">
                  <h3 className="font-medium text-white">{service.name}</h3>
                  <span
                    className={`rounded px-2 py-0.5 text-xs font-medium ${getStatusColor(service.status)}`}
                  >
                    {service.status.toUpperCase()}
                  </span>
                </div>

                <div className="mt-4 space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-400">CPU</span>
                    <span className="text-white">
                      {service.cpuPercent.toFixed(1)}%
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Memory</span>
                    <span className="text-white">
                      {service.memoryMB.toFixed(0)} MB
                      {service.memoryLimitMB > 0 &&
                        ` / ${service.memoryLimitMB.toFixed(0)} MB`}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Error Rate</span>
                    <span className="text-white">
                      {service.errorRate.toFixed(1)}/min
                    </span>
                  </div>
                  {service.name === "Scanner" && (
                    <div className="flex justify-between">
                      <span className="text-gray-400">WebSocket</span>
                      <span
                        className={
                          service.wsConnected
                            ? "text-green-400"
                            : "text-red-400"
                        }
                      >
                        {service.wsConnected ? "Connected" : "Disconnected"}
                      </span>
                    </div>
                  )}
                  {service.lastHeartbeat && (
                    <div className="flex justify-between">
                      <span className="text-gray-400">Last Heartbeat</span>
                      <span className="text-white">
                        {new Date(service.lastHeartbeat).toLocaleTimeString()}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            )
        )}
      </div>

      {health && health.alerts && health.alerts.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-white">Active Alerts</h3>
          <div className="space-y-2">
            {health.alerts.map((alert) => (
              <div
                key={alert.id}
                className="flex items-start gap-4 rounded-lg border border-gray-800 bg-gray-900 p-4"
              >
                <span
                  className={`rounded px-2 py-0.5 text-xs font-medium ${getSeverityColor(alert.severity)}`}
                >
                  {alert.severity.toUpperCase()}
                </span>
                <div className="flex-1">
                  <p className="text-white">{alert.message}</p>
                  <p className="mt-1 text-sm text-gray-400">
                    Triggered: {new Date(alert.triggeredAt).toLocaleString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {health && (
        <div className="text-sm text-gray-400">
          Last updated: {new Date(health.lastUpdated).toLocaleString()}
        </div>
      )}
    </div>
  );
}
