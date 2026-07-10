# Story 8.5: Frontend — Account Management UI (Epic 7)

Status: ready-for-dev

## Story

As a quant trader,
I want an account management UI to configure and manage multiple trading accounts,
so that I can switch between accounts and manage their settings.

## Acceptance Criteria

- [ ] Account list page
- [ ] Account creation form
- [ ] Account edit page
- [ ] Activate/deactivate actions
- [ ] API functions added to api.ts

## Tasks / Subtasks

- [ ] Task 1: Add API functions to api.ts
  - [ ] Subtask 1.1: Add account CRUD API functions
- [ ] Task 2: Add TypeScript types
  - [ ] Subtask 2.1: Add account types to types/index.ts
- [ ] Task 3: Create Account List Page
  - [ ] Subtask 3.1: Create account list with activate/deactivate
  - [ ] Subtask 3.2: Add navigation to admin layout
- [ ] Task 4: Create Account Form Page
  - [ ] Subtask 4.1: Create account creation form
  - [ ] Subtask 4.2: Create account edit form

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
