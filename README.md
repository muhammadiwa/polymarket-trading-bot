<div align="center">

# Polymarket Quant Arbitrage Platform (PQAP)

**Production-grade automated arbitrage trading platform for Polymarket prediction markets**

[![Build Status](https://github.com/muhammadiwa/polymarket-trading-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/muhammadiwa/polymarket-trading-bot/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26.4-00ADD8.svg?logo=go&logoColor=white)](https://golang.org/)
[![Python Version](https://img.shields.io/badge/Python-3.13-3776AB.svg?logo=python&logoColor=white)](https://python.org/)
[![Next.js](https://img.shields.io/badge/Next.js-16.2-000000.svg?logo=next.js&logoColor=white)](https://nextjs.org/)
[![Docker](https://img.shields.io/badge/Docker-24+-2496ED.svg?logo=docker&logoColor=white)](https://docker.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.36-326CE5.svg?logo=kubernetes&logoColor=white)](https://kubernetes.io/)

[Features](#features) • [Architecture](#architecture) • [Quick Start](#quick-start) • [API Reference](#api-reference) • [Deployment](#deployment) • [Contributing](#contributing) • [Troubleshooting](#troubleshooting)

</div>

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
  - [Epic 1: Foundation — Bot Can Hunt](#epic-1-foundation--bot-can-hunt)
  - [Epic 2: Risk Shield & Monitoring](#epic-2-risk-shield--monitoring)
  - [Epic 3: Advanced Hunting Strategies](#epic-3-advanced-hunting-strategies)
  - [Epic 4: Intelligence & Analytics](#epic-4-intelligence--analytics)
  - [Epic 5: Backtesting & Paper Trading](#epic-5-backtesting--paper-trading)
  - [Epic 6: AI Strategy Optimization](#epic-6-ai-strategy-optimization)
  - [Epic 7: Scaling & Enterprise](#epic-7-scaling--enterprise)
- [Architecture](#architecture)
  - [Design Paradigm](#design-paradigm)
  - [Service Architecture](#service-architecture)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
- [API Reference](#api-reference)
- [Project Structure](#project-structure)
- [Deployment](#deployment)
- [Monitoring](#monitoring)
- [Testing](#testing)
- [Contributing](#contributing)
- [Troubleshooting](#troubleshooting)
- [Security](#security)
- [License](#license)
- [Acknowledgments](#acknowledgments)
- [Support](#support)

---

## Overview

PQAP is a sophisticated quantitative trading platform designed to detect and execute arbitrage opportunities on Polymarket prediction markets. Built with an event-driven hexagonal architecture, it provides institutional-grade risk management, real-time monitoring, and AI-powered strategy optimization.

### Key Highlights

- **Sub-second opportunity detection** via WebSocket market data ingestion
- **Atomic YES+NO execution** with slippage protection
- **Centralized risk management** (Pit Boss) with configurable limits
- **Multi-account support** with per-account risk isolation
- **Real-time dashboard** with portfolio analytics
- **AI-powered strategy optimization** with pattern analysis

### Screenshots

> 📸 *Screenshots coming soon!*

<!-- 
![Dashboard Overview](docs/images/dashboard-overview.png)
![Risk Monitor](docs/images/risk-monitor.png)
![Analytics](docs/images/analytics.png)
-->

---

## Features

### Epic 1: Foundation — Bot Can Hunt
- WebSocket connection to Polymarket with automatic reconnection
- Real-time market catalog with stale detection
- Simple YES+NO arbitrage detection with opportunity scoring
- Limit order execution with slippage protection
- Position tracking with PnL calculation
- Risk management (daily budget, position limits)
- Telegram notifications for critical alerts

### Epic 2: Risk Shield & Monitoring
- Advanced risk controls (correlation limits, Batasi Win, metabolic rate)
- Real-time dashboard with portfolio overview
- Risk status monitoring with quick actions
- System health and opportunity feed
- Notification preferences with throttling

### Epic 3: Advanced Hunting Strategies
- Cross-market arbitrage detection
- Strategy manager with CRUD operations
- Multi-strategy isolation with capital allocation
- Portfolio strategy allocation with tier adjustment

### Epic 4: Intelligence & Analytics
- PnL performance metrics (daily, weekly, monthly)
- Charts and CSV export
- Anomaly detection for performance monitoring
- Trade history filtering and export
- Real-time orderbook viewer with depth chart

### Epic 5: Backtesting & Paper Trading
- Historical data replay with simulation
- Backtesting metrics and parameter sweeps
- Paper trading with separate PnL tracking
- Seamless switch between paper and live trading
- Replay mode with speed control

### Epic 6: AI Strategy Optimization
- Pattern analysis from trade history
- Parameter suggestions with expected impact
- A/B testing in paper trading
- Overfitting detection
- AI assistant for performance Q&A

### Epic 7: Scaling & Enterprise
- Multi-account wallet configuration
- Cross-account portfolio view
- Per-account risk limits
- Admin panel with system configuration
- Log viewer with filtering
- Database management (backup, restore, cleanup)

---

## Architecture

### Design Paradigm

PQAP follows an **event-driven hexagonal architecture** (ports & adapters pattern):

```
┌──────────────────────────────────────────────────────────┐
│                    DRIVERS (Adapters)                      │
│  Polymarket WS · REST · CLOB · Redis · PG · NATS · TG   │
├──────────────────────────────────────────────────────────┤
│                    PORTS (Interfaces)                      │
│  MarketDataPort · OrderPort · RiskPort · NotifyPort       │
│  StatePort · EventPort · MetricsPort                      │
├──────────────────────────────────────────────────────────┤
│                    DOMAIN CORE                             │
│  Scanner · ArbEngine · Execution · Position · Portfolio   │
│  Risk · Strategy · Analytics · Backtest                   │
├──────────────────────────────────────────────────────────┤
│                    APPLICATION SERVICES                    │
│  Orchestrator · Reconciler · CircuitBreaker · Scheduler   │
└──────────────────────────────────────────────────────────┘
```

**Why hexagonal?**
1. **Testability** — Every port can be mocked
2. **Swappability** — Swap adapters without touching core logic
3. **Loose coupling** — Components communicate via events
4. **Latency-aware** — Synchronous where speed matters (risk checks)

### Service Architecture

```mermaid
graph TB
    subgraph "Go Services (Execution-Critical)"
        Scanner[Scanner<br/>Market Data Ingestion]
        ArbEngine[Arb Engine<br/>Opportunity Detection]
        Execution[Execution Engine<br/>Order Placement]
        Position[Position Manager<br/>PnL Tracking]
        Risk[Risk Manager<br/>Pit Boss]
    end
    
    subgraph "Python Services (AI/Analytics)"
        Portfolio[Portfolio Manager<br/>Capital Management]
        Analytics[Analytics<br/>Performance Metrics]
        Backtest[Backtest Engine<br/>Strategy Testing]
        AIOptimizer[AI Optimizer<br/>Pattern Analysis]
        Notification[Notification<br/>Alert Delivery]
    end
    
    subgraph "Frontend"
        Dashboard[Next.js Dashboard<br/>Real-time UI]
        Admin[Admin Panel<br/>System Management]
    end
    
    subgraph "Infrastructure"
        NATS[NATS JetStream<br/>Event Bus]
        Redis[Redis 8.8<br/>Cache/State]
        PostgreSQL[PostgreSQL 17<br/>OLTP Database]
        TimescaleDB[TimescaleDB 2.x<br/>Time-Series]
        Prometheus[Prometheus 3.12<br/>Metrics]
        Grafana[Grafana 13<br/>Dashboards]
    end
    
    Scanner -->|MarketPriceUpdated| NATS
    ArbEngine -->|OpportunityDetected| NATS
    Execution -->|OrderFilled| NATS
    Risk -->|RiskAlert| NATS
    
    NATS --> ArbEngine
    NATS --> Execution
    NATS --> Dashboard
    
    Execution -->|Sync Check < 10ms| Risk
    Execution --> PostgreSQL
    Scanner --> Redis
    Risk --> Redis
    
    Dashboard -->|WebSocket| Admin
    Admin --> PostgreSQL
```

---

## Tech Stack

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| **Execution Runtime** | Go | 1.26.4 | Scanner, Arb Engine, Execution, Position, Risk |
| **AI/Analytics Runtime** | Python | 3.13.14 | Portfolio, Analytics, Backtest, AI Optimizer |
| **API Framework** | FastAPI | 0.139.0 | API Gateway |
| **Frontend** | Next.js | 16.2.10 (LTS) | Dashboard, Admin Panel |
| **Cache/Coordination** | Redis | 8.8.0 | Pit Boss state, market cache |
| **OLTP Database** | PostgreSQL | 17.10 | Trades, positions, strategies |
| **Time-Series** | TimescaleDB | 2.x | Market prices, opportunities |
| **Event Bus** | NATS | 2.10+ (JetStream) | Async event streaming |
| **Container** | Docker | 24+ | Containerization |
| **Orchestration** | Kubernetes | 1.36.2 | Container orchestration |
| **Metrics** | Prometheus | 3.12.0 | Metrics collection |
| **Dashboards** | Grafana | 13.0.3 | Operational monitoring |
| **Blockchain** | Polygon | Chain ID 137 | Polymarket settlement |
| **Stablecoin** | USDC (Polygon) | — | Trading collateral |

---

## Quick Start

### Prerequisites

- Docker 24+ and Docker Compose v2
- Go 1.26.4 (for local development)
- Python 3.13+ (for local development)
- Node.js 18+ (for dashboard development)
- PostgreSQL 17+ (or use Docker)
- Redis 8+ (or use Docker)

### Installation

```bash
# 1. Clone the repository
git clone https://github.com/muhammadiwa/polymarket-trading-bot.git
cd polymarket-trading-bot

# 2. Copy environment template
cp .env.example .env

# 3. Edit configuration (see Configuration section)
nano .env

# 4. Start all services
docker-compose up -d

# 5. Run database migrations
docker-compose exec api-gateway python -m alembic upgrade head

# 6. Access the dashboard
open http://localhost:3000
```

### Configuration

Create a `.env` file with the following variables:

```env
# ═══════════════════════════════════════════════════════════
# POLYMARKET API
# ═══════════════════════════════════════════════════════════
POLYMARKET_API_KEY=your_api_key_here
POLYMARKET_SECRET=your_api_secret_here

# ═══════════════════════════════════════════════════════════
# DATABASE
# ═══════════════════════════════════════════════════════════
POSTGRES_URL=postgres://pqap:password@localhost:5432/pqap
REDIS_URL=redis://localhost:6379

# ═══════════════════════════════════════════════════════════
# EVENT BUS
# ═══════════════════════════════════════════════════════════
NATS_URL=nats://localhost:4222

# ═══════════════════════════════════════════════════════════
# SECURITY (REQUIRED - Generate with: openssl rand -hex 32)
# ═══════════════════════════════════════════════════════════
JWT_SECRET=your_jwt_secret_here
ENCRYPTION_MASTER_KEY=your_encryption_key_here

# ═══════════════════════════════════════════════════════════
# NOTIFICATIONS (OPTIONAL)
# ═══════════════════════════════════════════════════════════
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
TELEGRAM_CHAT_ID=your_chat_id

# ═══════════════════════════════════════════════════════════
# RISK LIMITS (OPTIONAL - Override defaults)
# ═══════════════════════════════════════════════════════════
DEFAULT_DAILY_LOSS_LIMIT_PCT=2.0
DEFAULT_MAX_POSITION_PER_MARKET_PCT=10.0
DEFAULT_MAX_POSITION_PER_STRATEGY_PCT=20.0
DEFAULT_DRAWDOWN_THRESHOLD_PCT=10.0

# ═══════════════════════════════════════════════════════════
# APPLICATION
# ═══════════════════════════════════════════════════════════
LOG_LEVEL=info
ENVIRONMENT=development
```

---

## API Reference

### Authentication

All API endpoints require JWT authentication:

```bash
# Login to get token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your_password"}'

# Use token in requests
curl http://localhost:8080/api/portfolio/overview \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Core Endpoints

| Category | Method | Endpoint | Description |
|----------|--------|----------|-------------|
| **Portfolio** | `GET` | `/api/portfolio/overview` | Portfolio overview |
| **Portfolio** | `GET` | `/api/portfolio/overview?account_id={id}` | Per-account portfolio |
| **Positions** | `GET` | `/api/positions` | List all positions |
| **Risk** | `GET` | `/api/risk/status` | Risk status |
| **Risk** | `POST` | `/api/risk/emergency-stop` | Emergency stop |
| **Risk** | `POST` | `/api/risk/pause` | Pause trading |
| **Risk** | `POST` | `/api/risk/resume` | Resume trading |
| **Accounts** | `GET` | `/api/accounts` | List accounts |
| **Accounts** | `POST` | `/api/accounts` | Create account |

### Admin Endpoints

| Category | Method | Endpoint | Description |
|----------|--------|----------|-------------|
| **Config** | `GET` | `/api/admin/config` | List configurations |
| **Config** | `PUT` | `/api/admin/config/{key}` | Update config |
| **Health** | `GET` | `/api/admin/health` | System health |
| **Logs** | `GET` | `/api/admin/logs` | Query logs |
| **Database** | `POST` | `/api/admin/database/backup` | Create backup |
| **Database** | `GET` | `/api/admin/database/stats` | DB statistics |

### Risk Limits

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/risk/limits/{account_id}` | Get per-account limits |
| `PUT` | `/api/risk/limits/{account_id}` | Update limits (admin) |
| `GET` | `/api/risk/cross-account` | Cross-account exposure |

---

## Project Structure

```
polymarket-trading-bot/
├── services/
│   ├── scanner/                    # Go — Market data ingestion
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── websocket/          # Polymarket WS client
│   │   │   ├── rest/               # REST API client
│   │   │   └── catalog/            # Market catalog
│   │   └── adapters/
│   │
│   ├── arb-engine/                 # Go — Opportunity detection
│   │   ├── internal/
│   │   │   ├── detector/           # YES+NO, cross-market arb
│   │   │   ├── scorer/             # Opportunity scoring
│   │   │   └── filter/             # Threshold filtering
│   │   └── adapters/
│   │
│   ├── execution-engine/           # Go — Order execution
│   │   ├── internal/
│   │   │   ├── executor/           # Order placement
│   │   │   ├── monitor/            # Fill monitoring
│   │   │   └── circuit_breaker/    # API circuit breaker
│   │   └── adapters/
│   │
│   ├── risk-manager/               # Go — Pit Boss
│   │   ├── internal/
│   │   │   ├── pitboss/            # Risk evaluation
│   │   │   └── emergency/          # Emergency stop
│   │   └── adapters/
│   │
│   ├── position-manager/           # Go — Position tracking
│   ├── portfolio-manager/          # Python — Capital management
│   ├── analytics/                  # Python — Performance metrics
│   ├── backtest/                   # Python — Backtesting
│   ├── ai-optimizer/               # Python — AI optimization
│   ├── notification/               # Python — Telegram alerts
│   ├── api-gateway/                # Python (FastAPI) — API layer
│   ├── account-manager/            # Python — Multi-account
│   └── dashboard/                  # Next.js — Frontend
│       ├── src/app/                # App router pages
│       ├── src/components/         # React components
│       └── src/lib/                # Utilities & API client
│
├── migrations/
│   ├── postgres/                   # PostgreSQL migrations
│   └── timescale/                  # TimescaleDB migrations
│
├── monitoring/
│   ├── prometheus/prometheus.yaml
│   └── grafana/dashboards/         # Pre-built dashboards
│
├── deploy/
│   ├── docker/                     # Dockerfiles per service
│   └── k8s/                        # Kubernetes manifests
│
├── tests/
│   ├── unit/                       # Unit tests
│   ├── integration/                # Integration tests
│   └── e2e/                        # End-to-end tests
│
├── docker-compose.yaml             # Local development
├── Makefile                        # Build commands
├── CHANGELOG.md                    # Version history
└── README.md                       # This file
```

---

## Deployment

### Docker Compose (Development)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f scanner

# Stop all services
docker-compose down

# Rebuild specific service
docker-compose up -d --build scanner
```

### Kubernetes (Production)

```bash
# Create namespace
kubectl create namespace pqap

# Apply all manifests
kubectl apply -f deploy/k8s/

# Check status
kubectl get pods -n pqap

# View logs
kubectl logs -f deployment/scanner -n pqap

# Scale service
kubectl scale deployment/scanner --replicas=2 -n pqap
```

### Makefile Commands

```bash
make build          # Build all services
make test           # Run all tests
make lint           # Run linters
make proto          # Generate protobuf
make clean          # Clean build artifacts
make docker-build   # Build Docker images
make k8s-deploy     # Deploy to Kubernetes
```

---

## Monitoring

### Prometheus Metrics

All services export metrics on `/metrics`:

| Metric Prefix | Service | Description |
|---------------|---------|-------------|
| `pqap_scanner_*` | Scanner | Market data metrics |
| `pqap_execution_*` | Execution | Order metrics |
| `pqap_risk_*` | Risk Manager | Risk metrics |
| `pqap_portfolio_*` | Portfolio | Capital metrics |
| `pqap_admin_*` | Admin | Admin panel metrics |

### Grafana Dashboards

Pre-built dashboards in `monitoring/grafana/dashboards/`:

- **Trading Overview** — Portfolio PnL, positions, opportunities
- **Risk Monitor** — Risk limits, circuit breakers, alerts
- **System Health** — CPU, memory, connections, error rates
- **Strategy Performance** — Per-strategy metrics

### Alerts

| Severity | Examples |
|----------|----------|
| **Critical** | Emergency stop, circuit breaker tripped, API death spiral |
| **Warning** | High error rate, memory usage, connection issues |
| **Info** | Daily summary, performance milestones |

---

## Testing

```bash
# Unit tests
make test-unit

# Integration tests
make test-integration

# End-to-end tests
make test-e2e

# Coverage report
make test-coverage

# Specific service
cd services/api-gateway && pytest tests/
cd services/dashboard && npm test
```

---

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md).

### Quick Start

1. **Fork** the repository
2. **Clone** your fork
3. **Create** a feature branch
4. **Commit** with conventional commits
5. **Push** and open a Pull Request

### Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: resolve bug
docs: update documentation
style: formatting changes
refactor: code refactoring
test: add tests
chore: maintenance
```

### Code Style

| Language | Tool | Config |
|----------|------|--------|
| Go | `golangci-lint` | `.golangci.yml` |
| Python | `ruff` + `black` | `pyproject.toml` |
| TypeScript | `eslint` | `.eslintrc.js` |

### Pull Request Checklist

- [ ] Tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
- [ ] Conventional commit messages

---

## Troubleshooting

### Common Issues

#### Services won't start

```bash
# Check logs
docker-compose logs

# Verify environment variables
docker-compose config

# Restart specific service
docker-compose restart scanner
```

#### Database connection errors

```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Verify connection string
echo $POSTGRES_URL

# Run migrations
docker-compose exec api-gateway python -m alembic upgrade head
```

#### WebSocket disconnections

```bash
# Check scanner logs
docker-compose logs -f scanner

# Verify Polymarket API key
echo $POLYMARKET_API_KEY

# Check network connectivity
docker-compose exec scanner ping api.polymarket.com
```

#### High memory usage

```bash
# Check resource usage
docker stats

# Limit container memory
docker-compose up -d --scale scanner=1
```

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug
docker-compose up -d

# View detailed logs
docker-compose logs -f --tail=100
```

### Getting Help

1. Check [Issues](https://github.com/muhammadiwa/polymarket-trading-bot/issues)
2. Search [Discussions](https://github.com/muhammadiwa/polymarket-trading-bot/discussions)
3. Read [Documentation](docs/)

---

## Security

### Reporting Vulnerabilities

Please report security vulnerabilities to **security@your-org.com**. Do NOT open public issues.

### Security Measures

- ✅ JWT authentication with configurable expiry
- ✅ CSRF protection on state-changing endpoints
- ✅ Encrypted private key storage (AES-256-GCM)
- ✅ Rate limiting on sensitive endpoints
- ✅ Kubernetes secrets for sensitive config
- ✅ Input validation on all endpoints
- ✅ SQL injection prevention (parameterized queries)
- ✅ XSS protection (Content-Security-Policy)

---

## License

This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.

```
MIT License

Copyright (c) 2026 PQAP Team

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## Acknowledgments

- [Polymarket](https://polymarket.com) — Prediction market platform
- [NATS](https://nats.io) — High-performance messaging
- [TimescaleDB](https://www.timescale.com/) — Time-series database
- [FastAPI](https://fastapi.tiangolo.com/) — Modern Python web framework
- [Next.js](https://nextjs.org/) — React framework
- [Polygon](https://polygon.technology/) — Blockchain infrastructure

---

## Support

| Channel | Link |
|---------|------|
| 📖 Documentation | [docs/](docs/) |
| 🐛 Issues | [GitHub Issues](https://github.com/muhammadiwa/polymarket-trading-bot/issues) |
| 💬 Discussions | [GitHub Discussions](https://github.com/muhammadiwa/polymarket-trading-bot/discussions) |
| 📋 Changelog | [CHANGELOG.md](CHANGELOG.md) |
| 🔒 Security | security@your-org.com |

---

<div align="center">

**Built with ❤️ by the PQAP Team**

[⬆ Back to Top](#polymarket-quant-arbitrage-platform-pqap)

</div>
