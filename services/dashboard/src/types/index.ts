export interface PortfolioOverview {
  totalCapital: string;
  dailyPnL: string;
  totalPnL: string;
  utilizationRate: string;
  lastUpdated: string;
}

export interface Position {
  id: string;
  market: string;
  side: "YES" | "NO";
  entryPrice: string;
  currentPrice: string;
  quantity: string;
  unrealizedPnL: string;
  updatedAt: string;
}

export interface RiskStatus {
  dailyBudgetRemaining: string;
  dailyBudgetTotal: string;
  dailyBudgetUsedFraction: string;
  currentDrawdown: string;
  drawdownThreshold: string;
  winStreakCurrent: number;
  winStreakThreshold: number;
  circuitBreakerStatus: "open" | "closed";
  circuitBreakerTrippedAt: string | null;
  isPaused: boolean;
  pausedReason: string | null;
  lastUpdated: string;
}

export interface RiskParameterUpdate {
  dailyLossLimit?: string;
  maxPositionPerMarket?: string;
  maxPositionPerStrategy?: string;
}

export interface ServiceHealth {
  name: string;
  status: "up" | "down" | "degraded";
  wsConnected: boolean;
  cpuPercent: number;
  memoryMB: number;
  memoryLimitMB: number;
  errorRate: number;
  lastHeartbeat: string | null;
}

export interface SystemHealth {
  scanner: ServiceHealth;
  arbEngine: ServiceHealth;
  executionEngine: ServiceHealth;
  riskManager: ServiceHealth;
  positionManager: ServiceHealth;
  overall: "healthy" | "degraded" | "unhealthy";
  lastUpdated: string;
}

export interface Opportunity {
  id: string;
  market: string;
  marketSlug: string;
  score: string;
  spread: string;
  fillProbability: string;
  timestamp: string;
  status: "detected" | "executed" | "filtered";
  filterReason: string | null;
  executionLatencyMs: number | null;
}

export interface OpportunityListResponse {
  opportunities: Opportunity[];
  total_count: number;
  next_cursor: string | null;
}

export interface WSMessage {
  type: "portfolio_update" | "position_update" | "risk_update" | "health_update" | "opportunity";
  payload: PortfolioOverview | Position[] | RiskStatus | SystemHealth | Opportunity;
  timestamp: string;
}
