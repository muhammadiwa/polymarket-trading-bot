"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";

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
      const csrfRes = await fetch("/api/auth/csrf");
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

      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers,
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

      <style jsx>{`
        .login-root {
          position: relative;
          display: flex;
          min-height: 100vh;
          align-items: center;
          justify-content: center;
          padding: 1rem;
          overflow: hidden;
          background: #060a13;
        }

        /* ── Background atmosphere ── */
        .login-bg {
          position: absolute;
          inset: 0;
          overflow: hidden;
        }

        .login-grid {
          position: absolute;
          inset: 0;
          background-image:
            linear-gradient(rgba(0, 212, 255, 0.03) 1px, transparent 1px),
            linear-gradient(90deg, rgba(0, 212, 255, 0.03) 1px, transparent 1px);
          background-size: 60px 60px;
          mask-image: radial-gradient(ellipse 70% 60% at 50% 50%, black 30%, transparent 100%);
          -webkit-mask-image: radial-gradient(ellipse 70% 60% at 50% 50%, black 30%, transparent 100%);
        }

        /* ── Shimmer: traveling glow on grid lines ── */
        .login-shimmer {
          position: absolute;
          inset: 0;
          overflow: hidden;
          pointer-events: none;
        }

        /* Horizontal shimmer — thin bright line moving left to right */
        .shimmer-h {
          position: absolute;
          left: -20%;
          height: 1px;
          width: 20%;
          background: linear-gradient(
            90deg,
            transparent,
            rgba(0, 212, 255, 0.0) 10%,
            rgba(0, 212, 255, 0.15) 40%,
            rgba(0, 255, 136, 0.35) 50%,
            rgba(0, 212, 255, 0.15) 60%,
            rgba(0, 212, 255, 0.0) 90%,
            transparent
          );
          filter: blur(0.5px);
          animation: shimmerH 8s linear infinite;
        }

        .shimmer-h--1 { top: 15%; animation-duration: 7s; animation-delay: 0s; }
        .shimmer-h--2 { top: 32%; animation-duration: 9s; animation-delay: -2s; width: 15%; }
        .shimmer-h--3 { top: 50%; animation-duration: 6s; animation-delay: -4s; }
        .shimmer-h--4 { top: 68%; animation-duration: 10s; animation-delay: -1s; width: 18%; }
        .shimmer-h--5 { top: 85%; animation-duration: 8s; animation-delay: -5s; width: 12%; }

        @keyframes shimmerH {
          0% { left: -25%; opacity: 0; }
          5% { opacity: 1; }
          95% { opacity: 1; }
          100% { left: 105%; opacity: 0; }
        }

        /* Vertical shimmer — thin bright line moving top to bottom */
        .shimmer-v {
          position: absolute;
          top: -20%;
          width: 1px;
          height: 20%;
          background: linear-gradient(
            180deg,
            transparent,
            rgba(0, 212, 255, 0.0) 10%,
            rgba(0, 212, 255, 0.12) 40%,
            rgba(0, 255, 136, 0.3) 50%,
            rgba(0, 212, 255, 0.12) 60%,
            rgba(0, 212, 255, 0.0) 90%,
            transparent
          );
          filter: blur(0.5px);
          animation: shimmerV 9s linear infinite;
        }

        .shimmer-v--1 { left: 20%; animation-duration: 9s; animation-delay: -1s; }
        .shimmer-v--2 { left: 40%; animation-duration: 7s; animation-delay: -3s; height: 15%; }
        .shimmer-v--3 { left: 60%; animation-duration: 11s; animation-delay: 0s; }
        .shimmer-v--4 { left: 80%; animation-duration: 8s; animation-delay: -4s; height: 18%; }

        @keyframes shimmerV {
          0% { top: -25%; opacity: 0; }
          5% { opacity: 1; }
          95% { opacity: 1; }
          100% { top: 105%; opacity: 0; }
        }

        /* Sparkle dots — twinkling points at grid intersections */
        .shimmer-dot {
          position: absolute;
          width: 3px;
          height: 3px;
          border-radius: 50%;
          background: #00d4ff;
          box-shadow: 0 0 6px 2px rgba(0, 212, 255, 0.6), 0 0 12px 4px rgba(0, 255, 136, 0.3);
          animation: sparkle 3s ease-in-out infinite;
        }

        .shimmer-dot--1 { top: 20%; left: 25%; animation-delay: 0s; animation-duration: 2.5s; }
        .shimmer-dot--2 { top: 45%; left: 55%; animation-delay: -0.8s; animation-duration: 3.2s; }
        .shimmer-dot--3 { top: 70%; left: 35%; animation-delay: -1.5s; animation-duration: 2.8s; }
        .shimmer-dot--4 { top: 30%; left: 75%; animation-delay: -2s; animation-duration: 3.5s; }
        .shimmer-dot--5 { top: 60%; left: 15%; animation-delay: -0.3s; animation-duration: 2.2s; }

        @keyframes sparkle {
          0%, 100% { opacity: 0; transform: scale(0.5); }
          50% { opacity: 1; transform: scale(1.5); }
        }

        .login-glow {
          position: absolute;
          border-radius: 50%;
          filter: blur(100px);
          opacity: 0.4;
          animation: float 20s ease-in-out infinite;
        }

        .login-glow--1 {
          width: 500px;
          height: 500px;
          background: radial-gradient(circle, rgba(0, 212, 255, 0.15), transparent 70%);
          top: -10%;
          right: -5%;
          animation-delay: 0s;
        }

        .login-glow--2 {
          width: 400px;
          height: 400px;
          background: radial-gradient(circle, rgba(0, 255, 136, 0.1), transparent 70%);
          bottom: -10%;
          left: -5%;
          animation-delay: -7s;
        }

        .login-glow--3 {
          width: 300px;
          height: 300px;
          background: radial-gradient(circle, rgba(0, 212, 255, 0.08), transparent 70%);
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          animation-delay: -14s;
        }

        @keyframes float {
          0%, 100% { transform: translate(0, 0) scale(1); }
          33% { transform: translate(30px, -20px) scale(1.05); }
          66% { transform: translate(-20px, 15px) scale(0.95); }
        }

        /* ── Container ── */
        .login-container {
          position: relative;
          z-index: 1;
          width: 100%;
          max-width: 420px;
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 2rem;
          opacity: 0;
          transform: translateY(20px);
          transition: opacity 0.6s ease, transform 0.6s ease;
        }

        .login-container--visible {
          opacity: 1;
          transform: translateY(0);
        }

        /* ── Brand ── */
        .login-brand {
          text-align: center;
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 0.5rem;
        }

        .login-logo {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 56px;
          height: 56px;
          border-radius: 14px;
          background: rgba(0, 212, 255, 0.06);
          border: 1px solid rgba(0, 212, 255, 0.12);
          margin-bottom: 0.25rem;
        }

        .login-title {
          font-size: 1.75rem;
          font-weight: 800;
          letter-spacing: 0.15em;
          background: linear-gradient(135deg, #00d4ff, #00ff88);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
          margin: 0;
          line-height: 1.2;
        }

        .login-subtitle {
          font-size: 0.8rem;
          color: #475569;
          letter-spacing: 0.05em;
          margin: 0;
        }

        /* ── Card ── */
        .login-card {
          position: relative;
          width: 100%;
          border-radius: 16px;
          padding: 1px;
          background: linear-gradient(
            135deg,
            rgba(0, 212, 255, 0.2),
            rgba(0, 255, 136, 0.1),
            rgba(0, 212, 255, 0.05),
            rgba(0, 255, 136, 0.15)
          );
          background-size: 300% 300%;
          animation: borderShift 8s ease-in-out infinite;
        }

        @keyframes borderShift {
          0%, 100% { background-position: 0% 50%; }
          50% { background-position: 100% 50%; }
        }

        .login-card-border {
          position: absolute;
          inset: 0;
          border-radius: 16px;
          pointer-events: none;
        }

        .login-card-inner {
          background: #0d1117;
          border-radius: 15px;
          padding: 2rem;
        }

        .login-heading {
          font-size: 1.25rem;
          font-weight: 700;
          color: #f1f5f9;
          margin: 0 0 0.25rem;
        }

        .login-desc {
          font-size: 0.85rem;
          color: #64748b;
          margin: 0 0 1.75rem;
        }

        /* ── Form ── */
        .login-form {
          display: flex;
          flex-direction: column;
          gap: 1.25rem;
        }

        .login-field {
          display: flex;
          flex-direction: column;
          gap: 0.4rem;
        }

        .login-label {
          font-size: 0.8rem;
          font-weight: 500;
          color: #94a3b8;
          letter-spacing: 0.02em;
        }

        .login-input-wrap {
          position: relative;
          display: flex;
          align-items: center;
        }

        .login-input-icon {
          position: absolute;
          left: 14px;
          color: #475569;
          pointer-events: none;
          transition: color 0.2s;
        }

        .login-input {
          width: 100%;
          padding: 0.75rem 0.875rem 0.75rem 2.75rem;
          font-size: 0.9rem;
          color: #f1f5f9;
          background: #161b22;
          border: 1px solid #21262d;
          border-radius: 10px;
          outline: none;
          transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;
          font-family: inherit;
        }

        .login-input::placeholder {
          color: #475569;
        }

        .login-input:focus {
          border-color: rgba(0, 212, 255, 0.5);
          box-shadow: 0 0 0 3px rgba(0, 212, 255, 0.08);
          background: #1a1f27;
        }

        .login-input:focus ~ .login-input-icon,
        .login-input:focus + .login-input-icon {
          color: #00d4ff;
        }

        .login-input-wrap:focus-within .login-input-icon {
          color: #00d4ff;
        }

        .login-toggle-pw {
          position: absolute;
          right: 12px;
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 4px;
          background: none;
          border: none;
          color: #475569;
          cursor: pointer;
          border-radius: 6px;
          transition: color 0.2s, background 0.2s;
        }

        .login-toggle-pw:hover {
          color: #94a3b8;
          background: rgba(255, 255, 255, 0.05);
        }

        .login-toggle-pw:focus-visible {
          outline: 2px solid rgba(0, 212, 255, 0.5);
          outline-offset: 2px;
        }

        /* ── Error ── */
        .login-error {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.75rem 1rem;
          border-radius: 10px;
          background: rgba(255, 71, 87, 0.08);
          border: 1px solid rgba(255, 71, 87, 0.2);
          color: #f87171;
          font-size: 0.825rem;
          animation: shake 0.4s ease;
        }

        @keyframes shake {
          0%, 100% { transform: translateX(0); }
          20% { transform: translateX(-4px); }
          40% { transform: translateX(4px); }
          60% { transform: translateX(-2px); }
          80% { transform: translateX(2px); }
        }

        /* ── Button ── */
        .login-btn {
          width: 100%;
          padding: 0.8rem;
          font-size: 0.9rem;
          font-weight: 600;
          letter-spacing: 0.02em;
          color: #060a13;
          background: linear-gradient(135deg, #00d4ff, #00ff88);
          border: none;
          border-radius: 10px;
          cursor: pointer;
          transition: opacity 0.2s, transform 0.1s, box-shadow 0.2s;
          margin-top: 0.25rem;
          font-family: inherit;
        }

        .login-btn:hover:not(:disabled) {
          box-shadow: 0 4px 20px rgba(0, 212, 255, 0.25), 0 0 40px rgba(0, 255, 136, 0.1);
          transform: translateY(-1px);
        }

        .login-btn:active:not(:disabled) {
          transform: translateY(0);
        }

        .login-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .login-btn:focus-visible {
          outline: 2px solid #00d4ff;
          outline-offset: 2px;
        }

        .login-btn-loading {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 0.5rem;
        }

        .login-spinner {
          animation: spin 1s linear infinite;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        /* ── Footer ── */
        .login-footer {
          font-size: 0.7rem;
          color: #334155;
          display: flex;
          align-items: center;
          gap: 0.375rem;
          letter-spacing: 0.03em;
        }

        /* ── Reduced motion ── */
        @media (prefers-reduced-motion: reduce) {
          .login-glow,
          .login-card,
          .shimmer-h,
          .shimmer-v,
          .shimmer-dot {
            animation: none;
          }
          .login-container {
            opacity: 1;
            transform: none;
            transition: none;
          }
          .login-error {
            animation: none;
          }
          .login-spinner {
            animation: none;
          }
        }

        /* ── Mobile ── */
        @media (max-width: 480px) {
          .login-card-inner {
            padding: 1.5rem;
          }
          .login-title {
            font-size: 1.5rem;
          }
        }
      `}</style>
    </div>
  );
}
