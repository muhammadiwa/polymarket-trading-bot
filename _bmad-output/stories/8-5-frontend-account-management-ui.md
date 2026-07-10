# Story 8.5: Frontend — Account Management UI (Epic 7)

Status: in-progress

baseline_commit: current

## Story

As a quant trader,
I want an account management UI to configure and manage multiple trading accounts,
so that I can switch between accounts and manage their settings.

## Acceptance Criteria

- [x] Account list page
- [x] Account creation form
- [x] Account edit page
- [x] Activate/deactivate actions
- [x] API functions added to api.ts

## Tasks / Subtasks

- [x] Task 1: Add API functions to api.ts
  - [x] Subtask 1.1: Add account CRUD API functions
- [x] Task 2: Add TypeScript types
  - [x] Subtask 2.1: Add account types to types/index.ts
- [x] Task 3: Create Account List Page
  - [x] Subtask 3.1: Create account list with activate/deactivate
  - [x] Subtask 3.2: Add navigation to admin layout
- [x] Task 4: Create Account Form Page
  - [x] Subtask 4.1: Create account creation form
  - [x] Subtask 4.2: Create account edit form

## API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/accounts` | List all accounts |
| POST | `/api/accounts` | Create new account |
| GET | `/api/accounts/{id}` | Get account details |
| PUT | `/api/accounts/{id}` | Update account |
| DELETE | `/api/accounts/{id}` | Deactivate account |
| POST | `/api/accounts/{id}/activate` | Activate account |
| POST | `/api/accounts/{id}/deactivate` | Deactivate account |

## Dev Notes

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 — INF-3
- **Backend:** Account Manager service at `services/account-manager/`

### Data Models

```typescript
interface Account {
  id: string;
  name: string;
  walletAddress: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface AccountCreateRequest {
  name: string;
  walletAddress: string;
  privateKey: string;
}

interface AccountUpdateRequest {
  name?: string;
}

interface AccountListResponse {
  accounts: Account[];
  total: number;
}
```

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

**New Files:**
- `services/dashboard/src/app/admin/accounts/page.tsx` — Account list page
- `services/dashboard/src/app/admin/accounts/new/page.tsx` — Account creation form
- `services/dashboard/src/app/admin/accounts/[id]/page.tsx` — Account edit page

**Modified Files:**
- `services/dashboard/src/types/index.ts` — Added account types
- `services/dashboard/src/lib/api.ts` — Added account API functions
- `services/dashboard/src/app/admin/layout.tsx` — Added Accounts nav link
- `services/dashboard/src/app/admin/page.tsx` — Added Accounts card link
