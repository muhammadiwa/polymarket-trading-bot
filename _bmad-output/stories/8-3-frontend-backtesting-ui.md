# Story 8.3: Frontend — Backtesting UI (Epic 5)

Status: in-progress

baseline_commit: current

## Story

As a quant trader,
I want a backtesting UI to test strategies with historical data,
so that I can optimize parameters before deploying to live trading.

## Acceptance Criteria

- [x] Backtest page with run form
- [x] Results display with summary metrics
- [ ] Parameter sweep configuration
- [ ] Replay mode with play/pause/speed controls
- [x] API functions added to api.ts

## Tasks / Subtasks

- [x] Task 1: Add API functions to api.ts
  - [x] Subtask 1.1: Add backtest run function
  - [x] Subtask 1.2: Add backtest status/results functions
  - [x] Subtask 1.3: Add parameter sweep function
  - [x] Subtask 1.4: Add replay functions
- [x] Task 2: Add TypeScript types
  - [x] Subtask 2.1: Add backtest types to types/index.ts
- [x] Task 3: Create Backtest Page
  - [x] Subtask 3.1: Create backtest run form
  - [x] Subtask 3.2: Create results display
  - [x] Subtask 3.3: Add navigation to admin layout
- [ ] Task 4: Create Replay Page
  - [ ] Subtask 4.1: Create replay controls
  - [ ] Subtask 4.2: Add event stream display

## API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/backtesting/run` | Start backtest |
| GET | `/api/backtesting/{run_id}/status` | Get backtest status |
| GET | `/api/backtesting/{run_id}/results` | Get backtest results |
| GET | `/api/backtesting/{run_id}/report` | Get full report |
| POST | `/api/backtesting/sweep` | Run parameter sweep |
| POST | `/api/replay` | Start replay session |
| GET | `/api/replay/{session_id}/events` | Stream replay events (SSE) |
| POST | `/api/replay/{session_id}/step` | Step forward |
| POST | `/api/replay/{session_id}/speed` | Change speed |
| GET | `/api/replay/{session_id}/status` | Get replay status |
| DELETE | `/api/replay/{session_id}` | Stop replay |

## Dev Notes

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 — INF-3
- **Backend:** Backtesting service at `services/backtesting/`

### Data Models

```typescript
interface SimulationConfig {
  slippagePct: number;
  partialFillProbability: number;
  latencyMs: number;
  minFillRatio: number;
  rngSeed: number;
}

interface BacktestRequest {
  strategyId: string;
  startDate: string;
  endDate: string;
  simulation: SimulationConfig;
}

interface BacktestStatus {
  runId: string;
  status: string;
  progress?: string;
  startedAt?: string;
  completedAt?: string;
  errorMessage?: string;
}

interface BacktestSummary {
  totalPnl: string;
  totalTrades: number;
  winRate: string;
  sharpeRatio: string;
  maxDrawdown: string;
  profitFactor?: string;
  var95?: string;
}

interface BacktestTrade {
  timestamp: string;
  marketId: string;
  side: string;
  price: string;
  quantity: string;
  slippage: string;
  pnl: string;
  lookaheadWarning: boolean;
}

interface BacktestResults {
  summary: BacktestSummary;
  trades: BacktestTrade[];
  warnings: any[];
  dailyPnl?: any[];
}

interface SweepParameter {
  name: string;
  minValue: number;
  maxValue: number;
  step: number;
}

interface SweepRequest {
  strategyId: string;
  startDate: string;
  endDate: string;
  parameters: SweepParameter[];
  rankBy: string;
  simulation: SimulationConfig;
}

interface SweepResult {
  parameters: Record<string, number>;
  summary: BacktestSummary;
}

interface SweepResults {
  results: SweepResult[];
  best?: SweepResult;
  totalConfigs: number;
}
```

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

**New Files:**
- `services/dashboard/src/app/admin/backtest/page.tsx` — Backtest run form and results display

**Modified Files:**
- `services/dashboard/src/types/index.ts` — Added backtest types
- `services/dashboard/src/lib/api.ts` — Added backtest API functions
- `services/dashboard/src/app/admin/layout.tsx` — Added Backtest nav link
- `services/dashboard/src/app/admin/page.tsx` — Added Backtest card link
