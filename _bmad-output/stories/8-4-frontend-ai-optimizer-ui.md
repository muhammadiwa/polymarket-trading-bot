# Story 8.4: Frontend — AI Optimizer UI (Epic 6)

Status: ready-for-dev

## Story

As a quant trader,
I want an AI optimizer UI to analyze trade patterns and optimize strategy parameters,
so that I can improve trading performance based on data-driven insights.

## Acceptance Criteria

- [ ] Suggestions page with approve/reject actions
- [ ] A/B test monitoring page
- [ ] Overfitting analysis display
- [ ] API functions added to api.ts

## Tasks / Subtasks

- [ ] Task 1: Add API functions to api.ts
  - [ ] Subtask 1.1: Add optimizer API functions
  - [ ] Subtask 1.2: Add A/B test API functions
- [ ] Task 2: Add TypeScript types
  - [ ] Subtask 2.1: Add optimizer types to types/index.ts
- [ ] Task 3: Create Suggestions Page
  - [ ] Subtask 3.1: Create suggestions list with approve/reject
  - [ ] Subtask 3.2: Add navigation to admin layout
- [ ] Task 4: Create A/B Test Page
  - [ ] Subtask 4.1: Create A/B test monitoring view
  - [ ] Subtask 4.2: Add overfitting analysis display

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
