"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createWSClient, type WSStatus } from "@/lib/websocket";
import type { WSMessage, PortfolioOverview, Position, RiskStatus, SystemHealth, Opportunity } from "@/types";

type OpportunityListener = (opp: Opportunity) => void;

interface WSContextValue {
  portfolioData: PortfolioOverview | null;
  positionData: Position[] | null;
  riskData: RiskStatus | null;
  healthData: SystemHealth | null;
  wsStatus: WSStatus;
  onOpportunity: (listener: OpportunityListener) => () => void;
}

const WSContext = createContext<WSContextValue>({
  portfolioData: null,
  positionData: null,
  riskData: null,
  healthData: null,
  wsStatus: "disconnected",
  onOpportunity: () => () => {},
});

export function useWSContext() {
  return useContext(WSContext);
}

export function WSProvider({ children }: { children: ReactNode }) {
  const [portfolioData, setPortfolioData] = useState<PortfolioOverview | null>(null);
  const [positionData, setPositionData] = useState<Position[] | null>(null);
  const [riskData, setRiskData] = useState<RiskStatus | null>(null);
  const [healthData, setHealthData] = useState<SystemHealth | null>(null);
  const [wsStatus, setWsStatus] = useState<WSStatus>("disconnected");
  const clientRef = useRef<ReturnType<typeof createWSClient> | null>(null);
  const opportunityListeners = useRef<Set<OpportunityListener>>(new Set());

  const onOpportunity = useCallback((listener: OpportunityListener) => {
    opportunityListeners.current.add(listener);
    return () => {
      opportunityListeners.current.delete(listener);
    };
  }, []);

  useEffect(() => {
    clientRef.current = createWSClient({
      onMessage: (message: WSMessage) => {
        // #13: Basic payload validation
        if (!message || !message.type || !message.payload) {
          console.warn("[WS] Invalid message format:", message);
          return;
        }
        const payload = message.payload as Record<string, unknown>;
        if (message.type === "portfolio_update" && typeof payload.totalCapital === "string") {
          setPortfolioData(message.payload as PortfolioOverview);
        } else if (message.type === "position_update" && Array.isArray(payload)) {
          setPositionData(message.payload as Position[]);
        } else if (message.type === "risk_update" && typeof payload.isPaused === "boolean") {
          setRiskData(message.payload as RiskStatus);
        } else if (message.type === "health_update" && typeof payload.overall === "string") {
          setHealthData(message.payload as SystemHealth);
        } else if (message.type === "opportunity" && typeof payload.id === "string") {
          const opp = message.payload as Opportunity;
          for (const listener of opportunityListeners.current) {
            listener(opp);
          }
        }
      },
      onStatusChange: setWsStatus,
    });

    return () => {
      clientRef.current?.close();
    };
  }, []);

  const value = useMemo(() => ({
    portfolioData, positionData, riskData, healthData, wsStatus, onOpportunity
  }), [portfolioData, positionData, riskData, healthData, wsStatus, onOpportunity]);

  return (
    <WSContext.Provider value={value}>
      {children}
    </WSContext.Provider>
  );
}
