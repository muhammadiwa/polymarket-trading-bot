"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import "./login.css";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";
const MAX_ATTEMPTS = 5;
const COOLDOWN_MS = 30000;

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [mounted, setMounted] = useState(false);
  const [attempts, setAttempts] = useState(0);
  const [cooldownStartedAt, setCooldownStartedAt] = useState(0);
  const [cooldownRemaining, setCooldownRemaining] = useState(0);
  const usernameRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setMounted(true);
    usernameRef.current?.focus();
  }, []);

  useEffect(() => {
    if (cooldownRemaining <= 0) return;
    const interval = setInterval(() => {
      const remaining = Math.max(0, COOLDOWN_MS - (Date.now() - cooldownStartedAt));
      setCooldownRemaining(remaining);
      if (remaining <= 0) {
        setAttempts(0);
        clearInterval(interval);
      }
    }, 1000);
    return () => clearInterval(interval);
  }, [cooldownStartedAt, cooldownRemaining]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (attempts >= MAX_ATTEMPTS && cooldownRemaining > 0) {
      const remaining = Math.ceil(cooldownRemaining / 1000);
      setError(`Too many attempts. Try again in ${remaining}s.`);
      return;
    }

    setError(null);
    setLoading(true);

    try {
      const csrfRes = await fetch(`${API_BASE}/api/auth/csrf`, { credentials: "include" });
      let csrfToken = "";
      if (csrfRes.ok) {
        const csrfData = await csrfRes.json();
        csrfToken = csrfData.csrf_token ?? "";
      }

      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (csrfToken) {
        headers["X-CSRF-Token"] = csrfToken;
      }

      const res = await fetch(`${API_BASE}/api/auth/login`, {
        method: "POST",
        headers,
        credentials: "include",
        body: JSON.stringify({ username, password }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => null);
        throw new Error(data?.detail ?? "Invalid credentials");
      }

      router.push("/");
      router.refresh();
    } catch (err) {
      const newAttempts = attempts + 1;
      setAttempts(newAttempts);
      if (newAttempts >= MAX_ATTEMPTS) {
        setCooldownStartedAt(Date.now());
        setCooldownRemaining(COOLDOWN_MS);
      }
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-root">
      {/* Atmospheric background */}
      <div className="login-bg">
        <div className="login-grid" />

        {/* Shimmer lines — traveling glow along grid lines */}
        <div className="login-shimmer">
          {/* Horizontal shimmer lines */}
          <div className="shimmer-h shimmer-h--1" />
          <div className="shimmer-h shimmer-h--2" />
          <div className="shimmer-h shimmer-h--3" />
          <div className="shimmer-h shimmer-h--4" />
          <div className="shimmer-h shimmer-h--5" />
          {/* Vertical shimmer lines */}
          <div className="shimmer-v shimmer-v--1" />
          <div className="shimmer-v shimmer-v--2" />
          <div className="shimmer-v shimmer-v--3" />
          <div className="shimmer-v shimmer-v--4" />
          {/* Sparkle dots at intersections */}
          <div className="shimmer-dot shimmer-dot--1" />
          <div className="shimmer-dot shimmer-dot--2" />
          <div className="shimmer-dot shimmer-dot--3" />
          <div className="shimmer-dot shimmer-dot--4" />
          <div className="shimmer-dot shimmer-dot--5" />
        </div>

        <div className="login-glow login-glow--1" />
        <div className="login-glow login-glow--2" />
        <div className="login-glow login-glow--3" />
      </div>

      <div className={`login-container ${mounted ? "login-container--visible" : ""}`}>
        {/* Brand header */}
        <div className="login-brand">
          <div className="login-logo">
            <svg width="40" height="40" viewBox="0 0 40 40" fill="none" aria-hidden="true">
              <rect x="2" y="2" width="36" height="36" rx="8" stroke="url(#logo-grad)" strokeWidth="2" />
              <path d="M12 28L20 12L28 28" stroke="url(#logo-grad)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
              <line x1="15" y1="22" x2="25" y2="22" stroke="url(#logo-grad)" strokeWidth="2" strokeLinecap="round" />
              <defs>
                <linearGradient id="logo-grad" x1="0" y1="0" x2="40" y2="40">
                  <stop stopColor="#00d4ff" />
                  <stop offset="1" stopColor="#00ff88" />
                </linearGradient>
              </defs>
            </svg>
          </div>
          <h1 className="login-title">PQAP</h1>
          <p className="login-subtitle">Polymarket Quant Arbitrage Platform</p>
        </div>

        {/* Login card */}
        <div className="login-card">
          <div className="login-card-border" />
          <div className="login-card-inner">
            <h2 className="login-heading">Welcome back</h2>
            <p className="login-desc">Sign in to access your trading dashboard</p>

            <form onSubmit={handleSubmit} className="login-form">
              <div className="login-field">
                <label htmlFor="username" className="login-label">
                  Username
                </label>
                <div className="login-input-wrap">
                  <svg className="login-input-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                    <circle cx="12" cy="7" r="4" />
                  </svg>
                  <input
                    ref={usernameRef}
                    id="username"
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    required
                    autoComplete="username"
                    className="login-input"
                    placeholder="Enter your username"
                  />
                </div>
              </div>

              <div className="login-field">
                <label htmlFor="password" className="login-label">
                  Password
                </label>
                <div className="login-input-wrap">
                  <svg className="login-input-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                    <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                  </svg>
                  <input
                    id="password"
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    autoComplete="current-password"
                    className="login-input"
                    placeholder="Enter your password"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="login-toggle-pw"
                    aria-label={showPassword ? "Hide password" : "Show password"}
                    tabIndex={-1}
                  >
                    {showPassword ? (
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                        <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                        <line x1="1" y1="1" x2="23" y2="23" />
                      </svg>
                    ) : (
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                        <circle cx="12" cy="12" r="3" />
                      </svg>
                    )}
                  </button>
                </div>
              </div>

              {error && (
                <div className="login-error" role="alert">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <circle cx="12" cy="12" r="10" />
                    <line x1="15" y1="9" x2="9" y2="15" />
                    <line x1="9" y1="9" x2="15" y2="15" />
                  </svg>
                  <span>{error}</span>
                </div>
              )}

              <button
                type="submit"
                disabled={loading || !username || !password}
                className="login-btn"
              >
                {loading ? (
                  <span className="login-btn-loading">
                    <svg className="login-spinner" width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" strokeDasharray="32" strokeDashoffset="12" />
                    </svg>
                    Signing in...
                  </span>
                ) : (
                  "Sign in"
                )}
              </button>
            </form>
          </div>
        </div>

        {/* Footer */}
        <p className="login-footer">
          Secure connection &middot; Encrypted credentials
        </p>
      </div>

    </div>
  );
}
