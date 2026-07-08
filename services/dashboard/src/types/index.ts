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

export interface PnLByPeriod {
  date: string;
  pnl: string;
  trade_count: number;
}

export interface PnLByStrategy {
  strategy_id: string;
  total_pnl: string;
  trade_count: number;
}

export interface PnLByMarket {
  market_id: string;
  market_slug: string;
  total_pnl: string;
  trade_count: number;
}

export interface PnLData {
  by_period: PnLByPeriod[];
  by_strategy: PnLByStrategy[];
  by_market: PnLByMarket[];
  total_pnl: string;
  total_trades: number;
}

export interface HistogramData {
  pnls: string[];  // Decimal strings from backend
  count: number;
  bins: number;
}

export interface OrderbookLevel {
  price: string;
  size: string;
  cumulative: string;
}

export interface OrderbookSnapshot {
  market_id: string;
  bids: OrderbookLevel[];
  asks: OrderbookLevel[];
  spread: string;
  last_update: string;
}

export interface RecentTrade {
  price: string;
  size: string;
  side: string;
  timestamp: string;
}

// Admin Panel Types
export interface SystemConfig {
  id: string;
  configKey: string;
  configValue: any;
  category: "api_keys" | "risk_defaults" | "notification_settings";
  description: string | null;
  isSensitive: boolean;
  createdAt: string;
  updatedAt: string;
  updatedBy: string | null;
}

export interface SystemConfigListResponse {
  configs: SystemConfig[];
  total: number;
}

export interface SystemConfigUpdate {
  configValue: any;
  reason?: string;
  expectedUpdatedAt?: string;
}

export interface ConfigAuditLog {
  id: string;
  configKey: string;
  oldValue: any;
  newValue: any;
  changedBy: string;
  changedAt: string;
  reason: string | null;
}

export interface ConfigAuditLogListResponse {
  logs: ConfigAuditLog[];
  total: number;
}

export interface HealthAlert {
  id: string;
  service: string;
  metric: string;
  threshold: number;
  currentValue: number;
  severity: "warning" | "critical";
  triggeredAt: string;
  message: string;
}

export interface AdminHealthStatus extends SystemHealth {
  alerts: HealthAlert[];
}
