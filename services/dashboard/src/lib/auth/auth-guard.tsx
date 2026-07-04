"use client";

import { useEffect, useState, useCallback, createContext, useContext, useRef } from "react";
import { useRouter } from "next/navigation";

interface AuthUser {
  user_id: string;
  username: string;
  role: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  loading: boolean;
  logout: () => Promise<void>;
  fetchWithAuth: (url: string, init?: RequestInit) => Promise<Response>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const TOKEN_REFRESH_THRESHOLD = 0.8;

function parseJwtExp(token: string): number | null {
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const decoded = JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/")));
    return typeof decoded.exp === "number" ? decoded.exp : null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const logout = useCallback(async () => {
    try {
      await fetch("/api/auth/logout", { method: "POST" });
    } finally {
      setUser(null);
      router.push("/login");
    }
  }, [router]);

  const fetchCsrfToken = useCallback(async (): Promise<string> => {
    const res = await fetch("/api/auth/csrf");
    if (!res.ok) return "";
    const data = await res.json();
    return data.csrf_token ?? "";
  }, []);

  const fetchWithAuth = useCallback(
    async (url: string, init: RequestInit = {}): Promise<Response> => {
      const method = (init.method ?? "GET").toUpperCase();
      const headers: Record<string, string> = {
        ...Object.fromEntries(
          new Headers(init.headers).entries()
        ),
      };

      if (method !== "GET" && method !== "HEAD" && method !== "OPTIONS") {
        const csrfToken = await fetchCsrfToken();
        if (csrfToken) {
          headers["X-CSRF-Token"] = csrfToken;
        }
      }

      const res = await fetch(url, { ...init, headers });

      if (res.status === 401) {
        const refreshed = await tryRefresh();
        if (refreshed) {
          const freshCsrf = await fetchCsrfToken();
          if (freshCsrf) {
            headers["X-CSRF-Token"] = freshCsrf;
          }
          return fetch(url, { ...init, headers });
        }
        router.push("/login");
        throw new Error("Session expired");
      }

      return res;
    },
    [router, fetchCsrfToken]
  );

  const tryRefresh = useCallback(async (): Promise<boolean> => {
    try {
      const res = await fetch("/api/auth/refresh", { method: "POST" });
      if (!res.ok) return false;
      const data = await res.json();
      return !!data.access_token;
    } catch {
      return false;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function checkAuth() {
      try {
        const res = await fetch("/api/auth/me");
        if (!res.ok) {
          if (!cancelled) {
            setUser(null);
            setLoading(false);
          }
          return;
        }
        const data = await res.json();
        if (!cancelled) {
          setUser({
            user_id: data.id,
            username: data.username,
            role: data.role,
          });
          setLoading(false);
        }
      } catch {
        if (!cancelled) {
          setUser(null);
          setLoading(false);
        }
      }
    }

    checkAuth();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!user) return;

    function scheduleRefresh() {
      if (refreshTimerRef.current) {
        clearInterval(refreshTimerRef.current);
      }

      const cookies = document.cookie.split(";").map((c) => c.trim());
      const sessionCookie = cookies.find((c) => c.startsWith("pqap_session="));
      const token = sessionCookie?.split("=")[1];

      let delayMs = 24 * 60 * 60 * 1000 * TOKEN_REFRESH_THRESHOLD;
      if (token) {
        const exp = parseJwtExp(token);
        if (exp) {
          const expiresAt = exp * 1000;
          const now = Date.now();
          const ttl = expiresAt - now;
          if (ttl > 0) {
            delayMs = ttl * TOKEN_REFRESH_THRESHOLD;
          }
        }
      }

      refreshTimerRef.current = setInterval(
        async () => {
          const success = await tryRefresh();
          if (!success) {
            router.push("/login");
          } else {
            scheduleRefresh();
          }
        },
        Math.max(delayMs, 60_000)
      );
    }

    scheduleRefresh();

    return () => {
      if (refreshTimerRef.current) {
        clearInterval(refreshTimerRef.current);
      }
    };
  }, [user, router, tryRefresh]);

  return (
    <AuthContext.Provider value={{ user, loading, logout, fetchWithAuth }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading && !user) {
      router.push("/login");
    }
  }, [loading, user, router]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-950">
        <div className="text-gray-400">Loading...</div>
      </div>
    );
  }

  if (!user) {
    return null;
  }

  return <>{children}</>;
}

export function AdminGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading && !user) {
      router.push("/login");
    } else if (!loading && user && user.role !== "admin") {
      router.push("/");
    }
  }, [loading, user, router]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-950">
        <div className="text-gray-400">Loading...</div>
      </div>
    );
  }

  if (!user || user.role !== "admin") {
    return null;
  }

  return <>{children}</>;
}
