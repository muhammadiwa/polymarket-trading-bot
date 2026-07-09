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

// Log Viewer Types
export interface SystemLog {
  id: string;
  timestamp: string;
  level: "debug" | "info" | "warn" | "error" | "fatal";
  service: string;
  requestId: string | null;
  message: string;
  context: Record<string, any> | null;
}

export interface LogQueryParams {
  level?: string;
  service?: string;
  startDate?: string;
  endDate?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export interface LogQueryResponse {
  logs: SystemLog[];
  total: number;
  hasMore: boolean;
}

// Database Management Types
export interface BackupInfo {
  id: string;
  filename: string;
  filePath: string;
  sizeBytes: number;
  createdAt: string;
  status: "completed" | "failed" | "in_progress";
  durationMs: number | null;
  triggeredBy: string;
  errorMessage: string | null;
}

export interface BackupListResponse {
  backups: BackupInfo[];
  total: number;
}

export interface CleanupRequest {
  retentionDays: number;
  tables?: string[];
}

export interface CleanupResponse {
  deletedRows: Record<string, number>;
  freedBytes: number;
}

export interface DatabaseStats {
  totalSizeBytes: number;
  tableSizes: Record<string, number>;
  oldestLogTimestamp: string | null;
  newestLogTimestamp: string | null;
  totalLogEntries: number;
  totalTrades: number;
  totalPositions: number;
}

// Cross-Account Portfolio Types
export interface AccountPortfolioSummary {
  accountId: string;
  accountName: string;
  capital: string;
  dailyPnL: string;
  totalPnL: string;
  positionCount: number;
  utilizationRate: string;
  isActive: boolean;
}

export interface CrossAccountPortfolio {
  totalCapital: string;
  totalDailyPnL: string;
  totalPnL: string;
  totalPositions: number;
  accounts: AccountPortfolioSummary[];
  lastUpdated: string;
}

export interface PerAccountPortfolio {
  accountId: string;
  accountName: string;
  capital: string;
  dailyPnL: string;
  totalPnL: string;
  positions: Position[];
  utilizationRate: string;
  lastUpdated: string;
}

// Cross-Account Risk Types
export interface AccountRiskSummary {
  accountId: string;
  accountName: string;
  dailyLossLimit: string;
  dailyLossUsed: string;
  maxPositionPerMarket: string;
  currentExposure: string;
  status: 'healthy' | 'warning' | 'critical';
}

export interface CrossAccountRisk {
  totalExposure: string;
  totalDailyLoss: string;
  accounts: AccountRiskSummary[];
  overallStatus: 'healthy' | 'warning' | 'critical';
  lastUpdated: string;
}

export interface PerAccountRiskLimits {
  accountId: string;
  dailyLossLimit: string;
  maxPositionPerMarket: string;
  maxPositionPerStrategy: string;
  drawdownThreshold: string;
}

export interface RiskLimitsUpdate {
  dailyLossLimit?: string;
  maxPositionPerMarket?: string;
  maxPositionPerStrategy?: string;
  drawdownThreshold?: string;
}
