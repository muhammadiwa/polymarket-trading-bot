"use client";

import { useState, useRef, useCallback, useEffect } from "react";
import { AppShell } from "@/components/layout/AppShell";
import { ReplayControls } from "@/components/replay/ReplayControls";
import { DecisionLog } from "@/components/replay/DecisionLog";

interface Decision {
  market_id: string;
  detected: string;
  decision: string;
  reason: string;
  score: string;
  risk_result: string;
  timestamp: string;
}

export default function ReplayPage() {
  const [startDate, setStartDate] = useState(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 1);
    return d.toISOString().split("T")[0];
  });
  const [endDate, setEndDate] = useState(() => new Date().toISOString().split("T")[0]);
  const [speed, setSpeed] = useState(1);
  const [playing, setPlaying] = useState(false);
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const sessionIdRef = useRef<string | null>(null);

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (sessionIdRef.current) {
      // #3: Delete server session on cleanup
      const csrfToken = document.cookie.match(/(?:^|;\s*)pqap_csrf=([^;]*)/)?.[1];
      fetch(`/api/replay/${sessionIdRef.current}`, {
        method: "DELETE",
        credentials: "include",
        headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
      }).catch(() => {});
      sessionIdRef.current = null;
      setSessionId(null);
    }
  }, []);

  const startReplay = useCallback(async (speedOverride?: number) => {
    cleanup();
    setLoading(true);
    setError(null);
    setDecisions([]);

    const currentSpeed = speedOverride ?? speed;

    try {
      // Get CSRF token
      const csrfRes = await fetch("/api/auth/csrf");
      let csrfToken = "";
      if (csrfRes.ok) {
        const csrfData = await csrfRes.json();
        csrfToken = csrfData.csrf_token ?? "";
      }

      const res = await fetch("/api/replay", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
        },
        credentials: "include",
        body: JSON.stringify({ strategy_id: "default", start_date: startDate, end_date: endDate, speed: currentSpeed }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.detail || "Failed to start replay");
      }

      const data = await res.json();
      setSessionId(data.session_id);
      sessionIdRef.current = data.session_id;
      setPlaying(true);

      // #1: Use fetch with ReadableStream for auth support
      const tokenRes = await fetch("/api/replay/" + data.session_id + "/events", {
        credentials: "include",
      });

      if (!tokenRes.ok) {
        throw new Error("Failed to connect to replay stream");
      }

      const reader = tokenRes.body?.getReader();
      if (!reader) throw new Error("No stream reader");

      const decoder = new TextDecoder();
      let buffer = "";

      const processStream = async () => {
        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split("\n");
            buffer = lines.pop() || "";

            for (const line of lines) {
              if (line.startsWith("event: ")) {
                const eventType = line.slice(7).trim();
                // Next line should be data
              } else if (line.startsWith("data: ")) {
                try {
                  const data = JSON.parse(line.slice(6));
                  if (data.market_id) {
                    setDecisions((prev) => [...prev, { ...data, timestamp: data.timestamp || new Date().toISOString() }]);
                  }
                } catch {
                  // Ignore parse errors
                }
              }
            }
          }
        } finally {
          setPlaying(false);
        }
      };

      processStream();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to start replay");
      setPlaying(false);
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate, speed, cleanup]);

  const handlePlayPause = useCallback(() => {
    if (playing) {
      cleanup();
      setPlaying(false);
    } else {
      startReplay();
    }
  }, [playing, startReplay, cleanup]);

  // #2: Speed change restarts replay with new speed
  const handleSpeedChange = useCallback((newSpeed: number) => {
    setSpeed(newSpeed);
    if (playing && sessionIdRef.current) {
      // Update speed on server
      const csrfToken = document.cookie.match(/(?:^|;\s*)pqap_csrf=([^;]*)/)?.[1];
      fetch(`/api/replay/${sessionIdRef.current}/speed?speed=${newSpeed}`, {
        method: "POST",
        credentials: "include",
        headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
      }).catch(() => {});
    }
  }, [playing]);

  const handleStep = useCallback(async () => {
    if (!sessionIdRef.current) {
      // Start a new session first
      setLoading(true);
      try {
        // Get CSRF token
      const csrfRes = await fetch("/api/auth/csrf");
      let csrfToken = "";
      if (csrfRes.ok) {
        const csrfData = await csrfRes.json();
        csrfToken = csrfData.csrf_token ?? "";
      }

      const res = await fetch("/api/replay", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
          },
          credentials: "include",
          body: JSON.stringify({ strategy_id: "default", start_date: startDate, end_date: endDate, speed: 1 }),
        });
        if (!res.ok) throw new Error("Failed to start replay");
        const data = await res.json();
        sessionIdRef.current = data.session_id;
        setSessionId(data.session_id);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to start replay");
        setLoading(false);
        return;
      }
      setLoading(false);
    }

    try {
      const csrfToken = document.cookie.match(/(?:^|;\s*)pqap_csrf=([^;]*)/)?.[1];
      const res = await fetch(`/api/replay/${sessionIdRef.current}/step`, {
        method: "POST",
        credentials: "include",
        headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
      });
      const data = await res.json();
      if (data.events) {
        for (const event of data.events) {
          if (event.data?.market_id) {
            setDecisions((prev) => [...prev, { ...event.data, timestamp: event.timestamp }]);
          }
        }
      }
      if (!data.has_more) {
        setSessionId(null);
        sessionIdRef.current = null;
      }
    } catch (err) {
      console.error("Step failed:", err);
    }
  }, [startDate, endDate]);

  useEffect(() => {
    return cleanup;
  }, [cleanup]);

  return (
    <AppShell>
      <div className="space-y-6 p-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-white">Replay Mode</h1>
          <div className="flex items-center gap-3">
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
            />
            <span className="text-gray-400">to</span>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
            />
          </div>
        </div>

        <ReplayControls
          playing={playing}
          speed={speed}
          onPlayPause={handlePlayPause}
          onStep={handleStep}
          onSpeedChange={handleSpeedChange}
          disabled={loading}
        />

        {error && (
          <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 p-5 text-[#ff4757]">
            {error}
          </div>
        )}

        <DecisionLog decisions={decisions} />
      </div>
    </AppShell>
  );
}
