import type {
  OpportunityListResponse,
  PortfolioOverview,
  Position,
  RiskParameterUpdate,
  RiskStatus,
  SystemHealth,
  SystemConfig,
  SystemConfigListResponse,
  SystemConfigUpdate,
  ConfigAuditLogListResponse,
  AdminHealthStatus,
  LogQueryParams,
  LogQueryResponse,
  BackupInfo,
  BackupListResponse,
  CleanupResponse,
  DatabaseStats,
  CrossAccountPortfolio,
  PerAccountPortfolio,
  AccountPortfolioSummary,
  CrossAccountRisk,
  PerAccountRiskLimits,
  RiskLimitsUpdate,
  BacktestRequest,
  BacktestStatus,
  BacktestResults,
  SweepRequest,
  SweepResults,
  Suggestion,
  SuggestionListResponse,
  AnalysisResult,
  ABTest,
  ABTestResultSummary,
  OverfittingAnalysis,
  Account,
  AccountCreateRequest,
  AccountUpdateRequest,
  AccountListResponse,
} from "@/types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";
const DEFAULT_TIMEOUT_MS = 15_000;

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("jwt_token");
}

function getCsrfToken(): string | null {
  if (typeof window === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)pqap_csrf=([^;]*)/);
  return match ? match[1] : null;
}

// Convert camelCase to snake_case for API requests
// Handles consecutive uppercase correctly (e.g., "ABTestId" -> "ab_test_id")
function toSnakeCase(str: string): string {
  return str
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1_$2')
    .replace(/([a-z\d])([A-Z])/g, '$1_$2')
    .toLowerCase();
}

function convertKeysToSnakeCase(obj: any): any {
  if (obj === null || obj === undefined) return obj;
  if (Array.isArray(obj)) return obj.map(convertKeysToSnakeCase);
  if (typeof obj === 'object') {
    return Object.fromEntries(
      Object.entries(obj).map(([key, value]) => [toSnakeCase(key), convertKeysToSnakeCase(value)])
    );
  }
  return obj;
}

async function request<T>(path: string, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  const token = getToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      headers,
      signal: controller.signal,
      credentials: "include",
    });
    if (!res.ok) {
      if (res.status === 401) {
        window.location.href = "/login";
        throw new Error("Session expired");
      }
      // Try to parse error body for descriptive message
      const errorBody = await res.json().catch(() => null);
      throw new Error(errorBody?.detail ?? `API error: ${res.status} ${res.statusText}`);
    }
    const text = await res.text();
    if (!text) return undefined as T;
    try {
      return JSON.parse(text) as T;
    } catch {
      throw new Error("Invalid JSON response from server");
    }
  } finally {
    clearTimeout(timer);
  }
}

async function postRequest<T>(path: string, body?: unknown, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  const token = getToken();
  const csrfToken = getCsrfToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  if (csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      method: "POST",
      headers,
      body: body ? JSON.stringify(convertKeysToSnakeCase(body)) : undefined,
      signal: controller.signal,
      credentials: "include",
    });
    if (!res.ok) {
      if (res.status === 401) {
        window.location.href = "/login";
        throw new Error("Unauthorized");
      }
      const errorBody = await res.json().catch(() => null);
      throw new Error(errorBody?.detail ?? `API error: ${res.status} ${res.statusText}`);
    }
    return res.json();
  } finally {
    clearTimeout(timer);
  }
}

async function putRequest<T>(path: string, body: unknown, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  const token = getToken();
  const csrfToken = getCsrfToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  if (csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      method: "PUT",
      headers,
      body: JSON.stringify(convertKeysToSnakeCase(body)),
      signal: controller.signal,
      credentials: "include",
    });
    if (!res.ok) {
      if (res.status === 401) {
        window.location.href = "/login";
        throw new Error("Unauthorized");
      }
      const errorBody = await res.json().catch(() => null);
      throw new Error(errorBody?.detail ?? `API error: ${res.status} ${res.statusText}`);
    }
    return res.json();
  } finally {
    clearTimeout(timer);
  }
}

export async function fetchPortfolioOverview(): Promise<PortfolioOverview> {
  return request<PortfolioOverview>("/api/portfolio/overview");
}

export async function fetchPositions(): Promise<Position[]> {
  return request<Position[]>("/api/positions");
}

export async function fetchRiskStatus(): Promise<RiskStatus> {
  return request<RiskStatus>("/api/risk/status");
}

export async function fetchEmergencyStopToken(): Promise<{ confirmationToken: string }> {
  return postRequest<{ confirmationToken: string }>("/api/risk/emergency-stop/confirmationToken");
}

export async function triggerEmergencyStop(reason: string, confirmationToken: string): Promise<{ status: string }> {
  return postRequest<{ status: string }>("/api/risk/emergency-stop", { reason, confirmationToken });
}

export async function pauseTrading(reason?: string): Promise<{ status: string }> {
  return postRequest<{ status: string }>("/api/risk/pause", { reason });
}

export async function resumeTrading(): Promise<{ status: string }> {
  return postRequest<{ status: string }>("/api/risk/resume");
}

export async function updateRiskParameters(params: RiskParameterUpdate): Promise<{ status: string }> {
  return putRequest<{ status: string }>("/api/risk/parameters", params);
}

export async function fetchSystemHealth(): Promise<SystemHealth> {
  return request<SystemHealth>("/api/system/health");
}

export async function fetchOpportunities(cursor?: string, pageSize = 50, status?: string): Promise<OpportunityListResponse> {
  const params = new URLSearchParams();
  if (cursor) params.set("cursor", cursor);
  // #9: Validate pageSize bounds
  params.set("page_size", String(Math.min(Math.max(pageSize, 1), 200)));
  if (status && status !== "all") params.set("status", status);
  const qs = params.toString();
  return request<OpportunityListResponse>(`/api/opportunities${qs ? `?${qs}` : ""}`);
}

// Analytics API
export async function fetchAnalyticsPnL(startDate: string, endDate: string, groupBy = "day"): Promise<import("@/types").PnLData> {
  const params = new URLSearchParams({ start_date: startDate, end_date: endDate, group_by: groupBy });
  return request<import("@/types").PnLData>(`/api/analytics/pnl?${params.toString()}`);
}

export async function fetchAnalyticsHistogram(startDate: string, endDate: string, bins = 20): Promise<import("@/types").HistogramData> {
  const params = new URLSearchParams({ start_date: startDate, end_date: endDate, bins: String(bins) });
  return request<import("@/types").HistogramData>(`/api/analytics/histogram?${params.toString()}`);
}

export async function downloadCSV(startDate: string, endDate: string, side?: string, pnlSign?: string, strategyId?: string, marketId?: string): Promise<void> {
  const token = getToken();
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const params = new URLSearchParams({ start_date: startDate, end_date: endDate });
  if (side) params.set("side", side);
  if (pnlSign) params.set("pnl_sign", pnlSign);
  if (strategyId) params.set("strategy_id", strategyId);
  if (marketId) params.set("market_id", marketId);

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 30000);
  try {
    const res = await fetch(`${API_BASE}/api/analytics/export?${params.toString()}`, { headers, credentials: "include", signal: controller.signal });
    if (res.status === 401) {
      window.location.href = "/login";
      return;
    }
    if (!res.ok) throw new Error(`Export failed: ${res.status}`);

    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `trades_${startDate}_${endDate}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  } finally {
    clearTimeout(timer);
  }
}

export async function downloadJSON(startDate: string, endDate: string, side?: string, pnlSign?: string, strategyId?: string, marketId?: string): Promise<void> {
  const token = getToken();
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const params = new URLSearchParams({ start_date: startDate, end_date: endDate });
  if (side) params.set("side", side);
  if (pnlSign) params.set("pnl_sign", pnlSign);
  if (strategyId) params.set("strategy_id", strategyId);
  if (marketId) params.set("market_id", marketId);

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 30000);
  try {
    const res = await fetch(`${API_BASE}/api/analytics/export/json?${params.toString()}`, { headers, credentials: "include", signal: controller.signal });
    if (res.status === 401) {
      window.location.href = "/login";
      return;
    }
    if (!res.ok) throw new Error(`Export failed: ${res.status}`);

    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `trades_${startDate}_${endDate}.json`;
    a.click();
    URL.revokeObjectURL(url);
  } finally {
    clearTimeout(timer);
  }
}

// Orderbook API
export async function fetchOrderbook(marketId: string): Promise<import("@/types").OrderbookSnapshot> {
  return request<import("@/types").OrderbookSnapshot>(`/api/orderbook/${marketId}`);
}

export async function fetchRecentTrades(marketId: string, limit = 100): Promise<{ trades: import("@/types").RecentTrade[]; count: number }> {
  return request<{ trades: import("@/types").RecentTrade[]; count: number }>(`/api/orderbook/${marketId}/trades?limit=${limit}`);
}

// Admin Config API
export async function fetchAdminConfigs(category?: string): Promise<SystemConfigListResponse> {
  const params = category ? `?category=${category}` : "";
  return request<SystemConfigListResponse>(`/api/admin/config${params}`);
}

export async function fetchAdminConfig(key: string, unmask = false): Promise<SystemConfig> {
  const params = unmask ? "?unmask=true" : "";
  return request<SystemConfig>(`/api/admin/config/${key}${params}`);
}

export async function updateAdminConfig(key: string, update: SystemConfigUpdate): Promise<SystemConfig> {
  return putRequest<SystemConfig>(`/api/admin/config/${key}`, update);
}

export async function fetchConfigAuditLogs(key?: string, limit = 50, offset = 0): Promise<ConfigAuditLogListResponse> {
  const params = new URLSearchParams();
  if (key) params.set("key", key);
  params.set("limit", String(limit));
  params.set("offset", String(offset));
  return request<ConfigAuditLogListResponse>(`/api/admin/config/audit/logs?${params.toString()}`);
}

// Admin Health API
export async function fetchAdminHealth(): Promise<AdminHealthStatus> {
  return request<AdminHealthStatus>("/api/admin/health");
}

export async function fetchAdminHealthAlerts(): Promise<{ alerts: import("@/types").HealthAlert[]; total: number }> {
  return request<{ alerts: import("@/types").HealthAlert[]; total: number }>("/api/admin/health/alerts");
}

// Admin Log Viewer API
export async function fetchAdminLogs(params: LogQueryParams): Promise<LogQueryResponse> {
  const searchParams = new URLSearchParams();
  if (params.level) searchParams.set("level", params.level);
  if (params.service) searchParams.set("service", params.service);
  if (params.startDate) searchParams.set("start_date", params.startDate);
  if (params.endDate) searchParams.set("end_date", params.endDate);
  if (params.search) searchParams.set("search", params.search);
  searchParams.set("limit", String(params.limit || 100));
  searchParams.set("offset", String(params.offset || 0));
  return request<LogQueryResponse>(`/api/admin/logs?${searchParams.toString()}`);
}

export async function fetchLogServices(): Promise<string[]> {
  return request<string[]>("/api/admin/logs/services");
}

// Admin Database Management API
export async function createBackup(): Promise<BackupInfo> {
  return postRequest<BackupInfo>("/api/admin/database/backup");
}

export async function fetchBackups(): Promise<BackupListResponse> {
  return request<BackupListResponse>("/api/admin/database/backups");
}

export async function getRestoreConfirmToken(backupId: string): Promise<{ confirmationToken: string }> {
  return postRequest<{ confirmationToken: string }>(`/api/admin/database/restore/${backupId}/confirm-token`);
}

export async function restoreBackup(backupId: string, confirmationToken: string): Promise<{ status: string }> {
  return postRequest<{ status: string }>(`/api/admin/database/restore/${backupId}`, { confirmationToken });
}

export async function cleanupDatabase(retentionDays: number, tables?: string[]): Promise<CleanupResponse> {
  return postRequest<CleanupResponse>("/api/admin/database/cleanup", { retentionDays, tables });
}

export async function fetchDatabaseStats(): Promise<DatabaseStats> {
  return request<DatabaseStats>("/api/admin/database/stats");
}

// Cross-Account Portfolio API
export async function fetchCrossAccountPortfolio(): Promise<CrossAccountPortfolio> {
  return request<CrossAccountPortfolio>("/api/portfolio/cross-account");
}

export async function fetchPerAccountPortfolio(accountId: string): Promise<PerAccountPortfolio> {
  return request<PerAccountPortfolio>(`/api/portfolio/overview?account_id=${accountId}`);
}

export async function fetchAccountPortfolioSummaries(): Promise<AccountPortfolioSummary[]> {
  return request<AccountPortfolioSummary[]>("/api/portfolio/accounts");
}

// Cross-Account Risk API
export async function fetchCrossAccountRisk(): Promise<CrossAccountRisk> {
  return request<CrossAccountRisk>("/api/risk/cross-account");
}

export async function fetchPerAccountRisk(accountId: string): Promise<PerAccountRiskLimits> {
  return request<PerAccountRiskLimits>(`/api/risk/limits/${accountId}`);
}

export async function fetchRiskLimits(accountId: string): Promise<PerAccountRiskLimits> {
  return request<PerAccountRiskLimits>(`/api/risk/limits/${accountId}`);
}

export async function updateRiskLimits(accountId: string, limits: RiskLimitsUpdate): Promise<PerAccountRiskLimits> {
  return putRequest<PerAccountRiskLimits>(`/api/risk/limits/${accountId}`, limits);
}

// Backtesting API (Epic 5)
export async function startBacktest(req: BacktestRequest): Promise<BacktestStatus> {
  return postRequest<BacktestStatus>("/api/backtesting/run", req);
}

export async function fetchBacktestStatus(runId: string): Promise<BacktestStatus> {
  return request<BacktestStatus>(`/api/backtesting/${runId}/status`);
}

export async function fetchBacktestResults(runId: string): Promise<BacktestResults> {
  return request<BacktestResults>(`/api/backtesting/${runId}/results`);
}

export async function fetchBacktestReport(runId: string): Promise<import("@/types").BacktestResults> {
  return request<import("@/types").BacktestResults>(`/api/backtesting/${runId}/report`);
}

export async function runParameterSweep(req: SweepRequest): Promise<SweepResults> {
  return postRequest<SweepResults>("/api/backtesting/sweep", req);
}

// AI Optimizer API (Epic 6)
export async function runAnalysis(strategyId: string, minTrades = 100): Promise<AnalysisResult> {
  return postRequest<AnalysisResult>(`/api/optimizer/analyze?strategy_id=${strategyId}&min_trades=${minTrades}`);
}

export async function fetchSuggestions(strategyId?: string, status?: string, limit = 50, offset = 0): Promise<SuggestionListResponse> {
  const params = new URLSearchParams();
  if (strategyId) params.set("strategy_id", strategyId);
  if (status) params.set("status", status);
  params.set("limit", String(limit));
  params.set("offset", String(offset));
  return request<SuggestionListResponse>(`/api/optimizer/suggestions?${params.toString()}`);
}

export async function approveSuggestion(suggestionId: string): Promise<Suggestion> {
  return postRequest<Suggestion>(`/api/optimizer/suggestions/${suggestionId}/approve`);
}

export async function rejectSuggestion(suggestionId: string): Promise<Suggestion> {
  return postRequest<Suggestion>(`/api/optimizer/suggestions/${suggestionId}/reject`);
}

export async function startABTest(suggestionId: string, minSampleSize?: number): Promise<ABTest> {
  const body: any = {};
  if (minSampleSize) body.min_sample_size = minSampleSize;
  return postRequest<ABTest>(`/api/optimizer/suggestions/${suggestionId}/start-ab-test`, body);
}

export async function fetchABTest(abTestId: string): Promise<ABTest> {
  return request<ABTest>(`/api/optimizer/ab-tests/${abTestId}`);
}

export async function fetchABTestSummary(abTestId: string): Promise<ABTestResultSummary> {
  return request<ABTestResultSummary>(`/api/optimizer/ab-tests/${abTestId}/summary`);
}

export async function fetchOverfittingAnalysis(suggestionId: string): Promise<OverfittingAnalysis> {
  return request<OverfittingAnalysis>(`/api/optimizer/suggestions/${suggestionId}/overfitting-analysis`);
}

// Account Management API (Epic 7)
export async function fetchAccounts(isActive?: boolean): Promise<AccountListResponse> {
  const params = isActive !== undefined ? `?is_active=${isActive}` : "";
  return request<AccountListResponse>(`/api/accounts${params}`);
}

export async function fetchAccount(accountId: string): Promise<Account> {
  return request<Account>(`/api/accounts/${accountId}`);
}

export async function createAccount(request: AccountCreateRequest): Promise<Account> {
  return postRequest<Account>("/api/accounts", request);
}

export async function updateAccount(accountId: string, request: AccountUpdateRequest): Promise<Account> {
  return putRequest<Account>(`/api/accounts/${accountId}`, request);
}

export async function deactivateAccount(accountId: string): Promise<Account> {
  return postRequest<Account>(`/api/accounts/${accountId}/deactivate`);
}

export async function activateAccount(accountId: string): Promise<Account> {
  return postRequest<Account>(`/api/accounts/${accountId}/activate`);
}
