"use client";

import { useState, useRef, useCallback, useEffect } from "react";
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

  const startReplay = useCallback(async () => {
    setLoading(true);
    setError(null);
    setDecisions([]);

    try {
      const res = await fetch("/api/replay", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ strategy_id: "default", start_date: startDate, end_date: endDate, speed }),
      });

      if (!res.ok) {
        throw new Error("Failed to start replay");
      }

      const data = await res.json();
      setSessionId(data.session_id);
      setPlaying(true);

      // Start SSE stream
      const es = new EventSource(`/api/replay/${data.session_id}/events`);
      eventSourceRef.current = es;

      es.addEventListener("decision", (e) => {
        const decision = JSON.parse(e.data);
        setDecisions((prev) => [...prev, { ...decision, timestamp: new Date().toISOString() }]);
      });

      es.addEventListener("done", () => {
        setPlaying(false);
        es.close();
      });

      es.onerror = () => {
        setPlaying(false);
        es.close();
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to start replay");
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate, speed]);

  const handlePlayPause = useCallback(() => {
    if (playing) {
      eventSourceRef.current?.close();
      setPlaying(false);
    } else if (sessionId) {
      startReplay();
    } else {
      startReplay();
    }
  }, [playing, sessionId, startReplay]);

  const handleStep = useCallback(async () => {
    if (!sessionId) {
      await startReplay();
      return;
    }

    try {
      const res = await fetch(`/api/replay/${sessionId}/step`, {
        method: "POST",
        credentials: "include",
      });
      const data = await res.json();
      if (data.event) {
        setDecisions((prev) => [...prev, { ...data.event.data, timestamp: data.event.timestamp }]);
      }
      if (!data.has_more) {
        setSessionId(null);
      }
    } catch (err) {
      console.error("Step failed:", err);
    }
  }, [sessionId, startReplay]);

  useEffect(() => {
    return () => {
      eventSourceRef.current?.close();
    };
  }, []);

  return (
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
        onSpeedChange={setSpeed}
        disabled={loading}
      />

      {error && (
        <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 p-5 text-[#ff4757]">
          {error}
        </div>
      )}

      <DecisionLog decisions={decisions} />
    </div>
  );
}
