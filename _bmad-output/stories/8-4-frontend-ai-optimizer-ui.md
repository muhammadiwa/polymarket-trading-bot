# Story 8.4: Frontend — AI Optimizer UI (Epic 6)

Status: in-progress

baseline_commit: current

## Story

As a quant trader,
I want an AI optimizer UI to analyze trade patterns and optimize strategy parameters,
so that I can improve trading performance based on data-driven insights.

## Acceptance Criteria

- [x] Suggestions page with approve/reject actions
- [ ] A/B test monitoring page
- [x] Overfitting analysis display
- [x] API functions added to api.ts

## Tasks / Subtasks

- [x] Task 1: Add API functions to api.ts
  - [x] Subtask 1.1: Add optimizer API functions
  - [x] Subtask 1.2: Add A/B test API functions
- [x] Task 2: Add TypeScript types
  - [x] Subtask 2.1: Add optimizer types to types/index.ts
- [x] Task 3: Create Suggestions Page
  - [x] Subtask 3.1: Create suggestions list with approve/reject
  - [x] Subtask 3.2: Add navigation to admin layout
- [ ] Task 4: Create A/B Test Page
  - [ ] Subtask 4.1: Create A/B test monitoring view
  - [x] Subtask 4.2: Add overfitting analysis display

## API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/optimizer/analyze` | Run pattern analysis |
| GET | `/api/optimizer/suggestions` | List suggestions |
| POST | `/api/optimizer/suggestions/{id}/approve` | Approve suggestion |
| POST | `/api/optimizer/suggestions/{id}/reject` | Reject suggestion |
| POST | `/api/optimizer/suggestions/{id}/start-ab-test` | Start A/B test |
| GET | `/api/optimizer/ab-tests/{id}` | Get A/B test status |
| GET | `/api/optimizer/ab-tests/{id}/summary` | Get A/B test summary |
| GET | `/api/optimizer/suggestions/{id}/overfitting-analysis` | Get overfitting analysis |

## Dev Notes

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 — INF-3
- **Backend:** AI Optimizer service at `services/ai-optimizer/`

### Data Models

```typescript
interface Suggestion {
  id: string;
  strategyId: string;
  patternType: string;
  parameterName: string;
  currentValue: string;
  suggestedValue: string;
  expectedImpact: string;
  confidence: number;
  status: 'pending' | 'approved' | 'rejected';
  isOverfitting: boolean;
  degradationPct: string | null;
  createdAt: string;
  updatedAt: string;
}

interface SuggestionListResponse {
  suggestions: Suggestion[];
  total: number;
}

interface AnalysisResult {
  patternsFound: number;
  suggestionsGenerated: number;
  strategyId: string;
}

interface ABTest {
  id: string;
  suggestionId: string;
  strategyId: string;
  status: 'running' | 'completed' | 'failed';
  minSampleSize: number;
  currentSampleSize: number;
  pValue: string | null;
  meanDifference: string | null;
  recommendation: string | null;
  startedAt: string;
  completedAt: string | null;
}

interface ABTestResultSummary {
  abTestId: string;
  controlMean: string;
  treatmentMean: string;
  meanDifference: string;
  pValue: string;
  confidenceInterval: [string, string];
  recommendation: string;
  isSignificant: boolean;
}

interface OverfittingAnalysis {
  suggestionId: string;
  overfittingScore: number;
  inSampleWinRate: string;
  outOfSampleWinRate: string;
  degradationPct: string;
  isOverfitting: boolean;
  warning: string | null;
}
```

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
