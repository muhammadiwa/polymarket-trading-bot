"use client";

import { Card } from "@/components/ui/Card";
import { useSystemHealth } from "@/hooks/useSystemHealth";
import type { ServiceHealth, SystemHealth as SystemHealthType } from "@/types";

function statusColor(status: "up" | "down" | "degraded"): string {
  if (status === "up") return "text-[#00ff88]";
  if (status === "degraded") return "text-yellow-500";
  return "text-[#ff4757]";
}

function statusBgColor(status: "up" | "down" | "degraded"): string {
  if (status === "up") return "bg-[#00ff88]/10";
  if (status === "degraded") return "bg-yellow-500/10";
  return "bg-[#ff4757]/10";
}

function statusDotColor(status: "up" | "down" | "degraded"): string {
  if (status === "up") return "bg-[#00ff88]";
  if (status === "degraded") return "bg-yellow-500";
  return "bg-[#ff4757]";
}

function cpuColor(pct: number): string {
  if (pct < 60) return "text-[#00ff88]";
  if (pct < 80) return "text-yellow-500";
  return "text-[#ff4757]";
}

function cpuBarColor(pct: number): string {
  if (pct < 60) return "bg-[#00ff88]";
  if (pct < 80) return "bg-yellow-500";
  return "bg-[#ff4757]";
}

function errorRateColor(rate: number): string {
  if (rate < 1) return "text-[#00ff88]";
  if (rate < 5) return "text-yellow-500";
  return "text-[#ff4757]";
}

function overallStatusColor(overall: "healthy" | "degraded" | "unhealthy"): string {
  if (overall === "healthy") return "bg-[#00ff88]/10 text-[#00ff88]";
  if (overall === "degraded") return "bg-yellow-500/10 text-yellow-500";
  return "bg-[#ff4757]/10 text-[#ff4757]";
}

function ServiceCard({ service }: { service: ServiceHealth }) {
  return (
    <Card title={service.name}>
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs text-gray-400">Status</span>
          <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${statusColor(service.status)}`}>
            <span className={`h-2 w-2 rounded-full ${statusDotColor(service.status)}`} />
            {service.status.toUpperCase()}
          </span>
        </div>

        {service.wsConnected !== undefined && (
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-400">WebSocket</span>
            <span className={`text-xs font-medium ${service.wsConnected ? "text-[#00ff88]" : "text-[#ff4757]"}`}>
              {service.wsConnected ? "Connected" : "Disconnected"}
            </span>
          </div>
        )}

        <div>
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs text-gray-400">CPU</span>
            <span className={`text-xs font-mono font-medium ${cpuColor(service.cpuPercent)}`}>
              {service.cpuPercent.toFixed(1)}%
            </span>
          </div>
          <div className="h-1.5 rounded-full bg-white/10 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-500 ${cpuBarColor(service.cpuPercent)}`}
              style={{ width: `${Math.min(100, service.cpuPercent)}%` }}
            />
          </div>
        </div>

        <div className="flex items-center justify-between">
          <span className="text-xs text-gray-400">Memory</span>
          <span className="text-xs font-mono font-medium text-white">
            {service.memoryLimitMB > 0
              ? `${service.memoryMB.toFixed(0)} / ${service.memoryLimitMB.toFixed(0)} MB`
              : `${service.memoryMB.toFixed(0)} MB`}
          </span>
        </div>

        <div className="flex items-center justify-between">
          <span className="text-xs text-gray-400">Error Rate</span>
          <span className={`text-xs font-mono font-medium ${errorRateColor(service.errorRate)}`}>
            {service.errorRate.toFixed(1)}/min
          </span>
        </div>

        <div className="flex items-center justify-between">
          <span className="text-xs text-gray-400">Last Heartbeat</span>
          <span className="text-xs font-mono text-gray-300">
            {service.lastHeartbeat ? new Date(service.lastHeartbeat).toLocaleTimeString() : "—"}
          </span>
        </div>
      </div>
    </Card>
  );
}

function LoadingSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4" aria-busy="true" aria-label="Loading system health">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5 animate-pulse">
          <div className="h-4 w-24 bg-white/10 rounded mb-3" />
          <div className="space-y-3">
            <div className="h-3 w-full bg-white/10 rounded" />
            <div className="h-3 w-full bg-white/10 rounded" />
            <div className="h-3 w-full bg-white/10 rounded" />
            <div className="h-3 w-full bg-white/10 rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}

function overallLabel(overall: "healthy" | "degraded" | "unhealthy"): string {
  if (overall === "healthy") return "All Systems Operational";
  if (overall === "degraded") return "Degraded Performance";
  return "System Issues Detected";
}

const SERVICES: Array<{ key: keyof Omit<SystemHealthType, "overall" | "lastUpdated">; label: string }> = [
  { key: "scanner", label: "Scanner" },
  { key: "arbEngine", label: "Arb Engine" },
  { key: "executionEngine", label: "Execution Engine" },
  { key: "riskManager", label: "Risk Manager" },
  { key: "positionManager", label: "Position Manager" },
];

export function SystemHealth() {
  const { data, loading, error, wsStatus } = useSystemHealth();

  if (loading) {
    return (
      <section className="space-y-4" aria-label="System Health">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">System Health</h2>
        </div>
        <LoadingSkeleton />
      </section>
    );
  }

  if (error) {
    return (
      <section className="space-y-4" aria-label="System Health">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">System Health</h2>
        </div>
        <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 backdrop-blur-md p-5 text-[#ff4757]" role="alert">
          Failed to load system health: {error}
        </div>
      </section>
    );
  }

  if (!data) return null;

  return (
    <section className="space-y-4" aria-label="System Health">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">System Health</h2>
        <div className="flex items-center gap-3">
          <span
            className={`text-xs px-2 py-1 rounded-full ${overallStatusColor(data.overall)}`}
            role="status"
            aria-label={`Overall status: ${data.overall}`}
          >
            {overallLabel(data.overall)}
          </span>
          <span
            className={`text-xs px-2 py-1 rounded-full ${
              wsStatus === "connected"
                ? "bg-[#00ff88]/10 text-[#00ff88]"
                : wsStatus === "connecting"
                  ? "bg-yellow-500/10 text-yellow-500"
                  : "bg-[#ff4757]/10 text-[#ff4757]"
            }`}
            role="status"
            aria-label={`WebSocket status: ${wsStatus}`}
          >
            {wsStatus === "connected" ? "Live" : wsStatus === "connecting" ? "Connecting..." : "Offline"}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
        {SERVICES.map(({ key }) => (
          <ServiceCard key={key} service={data[key]} />
        ))}
      </div>
    </section>
  );
}
