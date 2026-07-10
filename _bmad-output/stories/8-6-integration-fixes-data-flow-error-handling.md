# Story 8.6: Integration Fixes — Data Flow & Error Handling

Status: ready-for-dev

## Story

As a developer,
I want to fix integration issues with data flow and error handling,
so that the system works correctly in multi-worker deployments and edge cases are handled.

## Acceptance Criteria

- [ ] Replay sessions stored in Redis
- [ ] Restore tokens stored in Redis
- [ ] Database backup uses PGPASSWORD env var
- [ ] NATS dead-letter queue configured
- [ ] Lookahead bias detection implemented
- [ ] Latency simulation implemented

## Tasks / Subtasks

- [ ] Task 1: Move replay sessions to Redis
  - [ ] Subtask 1.1: Update replay.py to use Redis for session storage
- [ ] Task 2: Move restore tokens to Redis
  - [ ] Subtask 2.1: Update database.py to use Redis for tokens
- [ ] Task 3: Fix database backup security
  - [ ] Subtask 3.1: Use PGPASSWORD env var instead of URL
- [ ] Task 4: Implement lookahead bias detection
  - [ ] Subtask 4.1: Implement _detect_lookahead function

## API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/replay` | Start replay session (Redis-backed) |
| POST | `/api/admin/database/backup` | Secure backup |

## Dev Notes

### Architecture Context

- **Backend:** Python services with FastAPI
- **Cache:** Redis 8.8.0 for session storage
- **Database:** PostgreSQL 17.10

## Dev Agent Record

### File List
