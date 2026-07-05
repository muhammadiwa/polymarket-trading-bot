import type {
  OpportunityListResponse,
  PortfolioOverview,
  Position,
  RiskParameterUpdate,
  RiskStatus,
  SystemHealth,
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
        return undefined as T; // #8: return instead of throw during redirect
      }
      throw new Error(`API error: ${res.status} ${res.statusText}`);
    }
    // #14: Guard against non-JSON responses
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

  const csrfToken = getCsrfToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      method: "POST",
      headers,
      body: body ? JSON.stringify(body) : undefined,
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

  const csrfToken = getCsrfToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      method: "PUT",
      headers,
      body: JSON.stringify(body),
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
