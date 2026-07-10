# Story 8.7: Security Hardening

Status: ready-for-dev

## Story

As a developer,
I want to harden security across all services,
so that the system is protected against common vulnerabilities.

## Acceptance Criteria

- [ ] position-manager requires JWT secret (fail fast)
- [ ] CSRF middleware added to account-manager
- [ ] API prefixes standardized

## Tasks / Subtasks

- [ ] Task 1: Fix position-manager JWT validation
  - [ ] Subtask 1.1: Add JWT_SECRET validation in config
- [ ] Task 2: Add CSRF to account-manager
  - [ ] Subtask 2.1: Add CSRF middleware
- [ ] Task 3: Standardize API prefixes
  - [ ] Subtask 3.1: Update trades router prefix

## Dev Notes

### Architecture Context

- **Security:** JWT auth, CSRF protection — AD-14
- **Pattern:** All services must validate JWT_SECRET at startup

## Dev Agent Record

### File List
