"use client";

import { useState, useRef } from "react";
import { useRouter } from "next/navigation";

const MAX_ATTEMPTS = 5;
const COOLDOWN_MS = 30000;

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const attemptsRef = useRef(0);
  const cooldownRef = useRef(0);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    // Client-side rate limiting
    const now = Date.now();
    if (attemptsRef.current >= MAX_ATTEMPTS && now - cooldownRef.current < COOLDOWN_MS) {
      const remaining = Math.ceil((COOLDOWN_MS - (now - cooldownRef.current)) / 1000);
      setError(`Too many attempts. Try again in ${remaining} seconds.`);
      return;
    }
    if (now - cooldownRef.current >= COOLDOWN_MS) {
      attemptsRef.current = 0;
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
      attemptsRef.current++;
      cooldownRef.current = Date.now();
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-950 px-4">
      <div className="w-full max-w-sm rounded-lg border border-gray-800 bg-gray-900 p-8">
        <h1 className="mb-6 text-center text-2xl font-bold text-white">
          PQAP Login
        </h1>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="username"
              className="mb-1 block text-sm font-medium text-gray-300"
            >
              Username
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              autoComplete="username"
              className="w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-white placeholder-gray-500 focus:border-blue-500 focus:outline-none"
              placeholder="Enter username"
            />
          </div>

          <div>
            <label
              htmlFor="password"
              className="mb-1 block text-sm font-medium text-gray-300"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
              className="w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-white placeholder-gray-500 focus:border-blue-500 focus:outline-none"
              placeholder="Enter password"
            />
          </div>

          {error && (
            <div className="rounded bg-red-900/50 px-3 py-2 text-sm text-red-300">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? "Signing in..." : "Sign in"}
          </button>
        </form>
      </div>
    </div>
  );
}
