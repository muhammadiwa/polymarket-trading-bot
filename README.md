<![CDATA[<div align="center">

# Polymarket Quant Arbitrage Platform (PQAP)

**Production-grade automated arbitrage trading platform for Polymarket prediction markets**

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Go Version](https://img.shields.io/badge/Go-1.26.4-00ADD8?logo=go)]()
[![Python Version](https://img.shields.io/badge/Python-3.13-3776AB?logo=python)]()
[![Next.js](https://img.shields.io/badge/Next.js-16.2-000000?logo=next.js)]()
[![Docker](https://img.shields.io/badge/Docker-24+-2496ED?logo=docker)]()
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.36-326CE5?logo=kubernetes)]()

[Features](#features) • [Architecture](#architecture) • [Quick Start](#quick-start) • [API Reference](#api-reference) • [Deployment](#deployment) • [Contributing](#contributing)

</div>

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

### Service Architecture

```mermaid
graph TB
    subgraph "Go Services"
        Scanner[Scanner<br/>Market Data]
        ArbEngine[Arb Engine<br/>Opportunity Detection]
        Execution[Execution Engine<br/>Order Placement]
        Position[Position Manager<br/>PnL Tracking]
        Risk[Risk Manager<br/>Pit Boss]
    end
    
    subgraph "Python Services"
        Portfolio[Portfolio Manager<br/>Capital Management]
        Analytics[Analytics<br/>Performance Metrics]
        Backtest[Backtest Engine<br/>Strategy Testing]
        AIOptimizer[AI Optimizer<br/>Pattern Analysis]
        Notification[Notification<br/>Alert Delivery]
    end
    
    subgraph "Frontend"
        Dashboard[Next.js Dashboard<br/>Real-time UI]
    end
    
    subgraph "Infrastructure"
        NATS[NATS<br/>Event Bus]
        Redis[Redis<br/>Cache/State]
        PostgreSQL[PostgreSQL<br/>OLTP]
        TimescaleDB[TimescaleDB<br/>Time-Series]
        Prometheus[Prometheus<br/>Metrics]
    end
    
    Scanner -->|MarketPriceUpdated| NATS
    ArbEngine -->|OpportunityDetected| NATS
    Execution -->|OrderFilled| NATS
    Risk -->|RiskAlert| NATS
    
    NATS --> ArbEngine
    NATS --> Execution
    NATS --> Dashboard
    
    Execution -->|Sync Check| Risk
    Execution --> PostgreSQL
    Scanner --> Redis
    Risk --> Redis
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
| **Event Bus** | NATS | 2.10+ | Async event streaming |
| **Container** | Docker | 24+ | Containerization |
| **Orchestration** | Kubernetes | 1.36.2 | Container orchestration |
| **Metrics** | Prometheus | 3.12.0 | Metrics collection |
| **Dashboards** | Grafana | 13.0.3 | Operational monitoring |

---

## Quick Start

### Prerequisites

- Docker 24+ and Docker Compose
- Go 1.26.4 (for local development)
- Python 3.13+ (for local development)
- Node.js 18+ (for dashboard development)
- PostgreSQL 17+ (or use Docker)
- Redis 8+ (or use Docker)

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/polymarket-trading-bot.git
cd polymarket-trading-bot
```

### 2. Environment Configuration

```bash
# Copy example environment file
cp .env.example .env

# Edit with your configuration
nano .env
```

Required environment variables:

```env
# Polymarket API
POLYMARKET_API_KEY=your_api_key
POLYMARKET_SECRET=your_api_secret

# Database
POSTGRES_URL=postgres://user:pass@localhost:5432/pqap
REDIS_URL=redis://localhost:6379

# NATS
NATS_URL=nats://localhost:4222

# Security
JWT_SECRET=your_jwt_secret
ENCRYPTION_MASTER_KEY=your_encryption_key

# Notifications
TELEGRAM_BOT_TOKEN=your_telegram_token
TELEGRAM_CHAT_ID=your_chat_id
```

### 3. Start with Docker Compose

```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f scanner
```

### 4. Run Database Migrations

```bash
# Run PostgreSQL migrations
docker-compose exec api-gateway python -m alembic upgrade head

# Or using golang-migrate
migrate -path migrations/postgres -database $POSTGRES_URL up
```

### 5. Access the Dashboard

Open [http://localhost:3000](http://localhost:3000) in your browser.

Default credentials:
- Username: `admin`
- Password: Set via `ADMIN_PASSWORD` environment variable

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_URL` | PostgreSQL connection string | `postgres://localhost:5432/pqap` |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379` |
| `NATS_URL` | NATS connection string | `nats://localhost:4222` |
| `JWT_SECRET` | JWT signing secret | (required) |
| `ENCRYPTION_MASTER_KEY` | Master key for wallet encryption | (required) |
| `POLYMARKET_API_KEY` | Polymarket API key | (required) |
| `POLYMARKET_SECRET` | Polymarket API secret | (required) |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | (optional) |
| `TELEGRAM_CHAT_ID` | Telegram chat ID | (optional) |
| `LOG_LEVEL` | Logging level | `info` |
| `ENVIRONMENT` | Environment (development/production) | `development` |

### Risk Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `DEFAULT_DAILY_LOSS_LIMIT_PCT` | Daily loss limit (%) | `2.0` |
| `DEFAULT_MAX_POSITION_PER_MARKET_PCT` | Max position per market (%) | `10.0` |
| `DEFAULT_MAX_POSITION_PER_STRATEGY_PCT` | Max position per strategy (%) | `20.0` |
| `DEFAULT_DRAWDOWN_THRESHOLD_PCT` | Drawdown circuit breaker (%) | `10.0` |

---

## API Reference

### Authentication

All API endpoints require JWT authentication via `Authorization: Bearer <token>` header.

### Portfolio Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/portfolio/overview` | Get portfolio overview |
| `GET` | `/api/portfolio/overview?account_id={id}` | Per-account portfolio |
| `GET` | `/api/positions` | List all positions |

### Risk Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/risk/status` | Get risk status |
| `POST` | `/api/risk/emergency-stop` | Trigger emergency stop |
| `POST` | `/api/risk/pause` | Pause trading |
| `POST` | `/api/risk/resume` | Resume trading |
| `GET` | `/api/risk/limits/{account_id}` | Get per-account risk limits |
| `PUT` | `/api/risk/limits/{account_id}` | Update risk limits |

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/admin/config` | List system configurations |
| `PUT` | `/api/admin/config/{key}` | Update configuration |
| `GET` | `/api/admin/health` | System health with alerts |
| `GET` | `/api/admin/logs` | Query system logs |
| `POST` | `/api/admin/database/backup` | Create database backup |
| `GET` | `/api/admin/database/stats` | Database statistics |

### Account Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/accounts` | List all accounts |
| `POST` | `/api/accounts` | Create new account |
| `GET` | `/api/accounts/{id}` | Get account details |
| `PUT` | `/api/accounts/{id}` | Update account |
| `DELETE` | `/api/accounts/{id}` | Deactivate account |

---

## Project Structure

```
polymarket-trading-bot/
├── services/
│   ├── scanner/                    # Go — Market data ingestion
│   ├── arb-engine/                 # Go — Opportunity detection
│   ├── execution-engine/           # Go — Order execution
│   ├── position-manager/           # Go — Position tracking
│   ├── risk-manager/               # Go — Risk management
│   ├── portfolio-manager/          # Python — Capital management
│   ├── analytics/                  # Python — Performance analytics
│   ├── backtest/                   # Python — Backtesting engine
│   ├── ai-optimizer/               # Python — AI strategy optimization
│   ├── notification/               # Python — Alert delivery
│   ├── api-gateway/                # Python (FastAPI) — API layer
│   ├── account-manager/            # Python — Multi-account management
│   └── dashboard/                  # Next.js — Frontend
│
├── migrations/
│   ├── postgres/                   # PostgreSQL migrations
│   └── timescale/                  # TimescaleDB migrations
│
├── monitoring/
│   ├── prometheus/                 # Prometheus configuration
│   └── grafana/                    # Grafana dashboards
│
├── deploy/
│   ├── docker/                     # Dockerfiles
│   └── k8s/                        # Kubernetes manifests
│
├── tests/
│   ├── unit/                       # Unit tests
│   ├── integration/                # Integration tests
│   └── e2e/                        # End-to-end tests
│
├── docker-compose.yaml             # Local development
├── Makefile                        # Build, test, lint commands
└── README.md                       # This file
```

---

## Deployment

### Docker Compose (Development)

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# Rebuild and restart
docker-compose up -d --build
```

### Kubernetes (Production)

```bash
# Create namespace
kubectl create namespace pqap

# Apply configurations
kubectl apply -f deploy/k8s/

# Check deployment status
kubectl get pods -n pqap

# View logs
kubectl logs -f deployment/scanner -n pqap
```

### Makefile Commands

```bash
# Build all services
make build

# Run tests
make test

# Run linter
make lint

# Generate protobuf
make proto

# Clean build artifacts
make clean
```

---

## Monitoring

### Prometheus Metrics

All services export metrics on `/metrics` endpoint:

- `pqap_scanner_*` — Scanner metrics
- `pqap_execution_*` — Execution metrics
- `pqap_risk_*` — Risk management metrics
- `pqap_portfolio_*` — Portfolio metrics

### Grafana Dashboards

Pre-built dashboards available in `monitoring/grafana/dashboards/`:

- **Trading Overview** — Portfolio PnL, positions, opportunities
- **Risk Monitor** — Risk limits, circuit breakers, alerts
- **System Health** — CPU, memory, connections, error rates
- **Strategy Performance** — Per-strategy metrics and comparison

### Alerts

Alertmanager configured for:

- Critical: Emergency stop, circuit breaker tripped
- Warning: High error rate, memory usage, connection issues
- Info: Daily summary, performance milestones

---

## Testing

### Unit Tests

```bash
# Go services
go test ./...

# Python services
pytest services/api-gateway/tests/
pytest services/portfolio-manager/tests/

# Frontend
cd services/dashboard && npm test
```

### Integration Tests

```bash
# Run integration tests
make test-integration
```

### End-to-End Tests

```bash
# Run E2E tests
make test-e2e
```

---

## Contributing

We welcome contributions! Please follow these guidelines:

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Commit Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — New feature
- `fix:` — Bug fix
- `docs:` — Documentation
- `style:` — Formatting
- `refactor:` — Code refactoring
- `test:` — Adding tests
- `chore:` — Maintenance

### Code Style

- **Go**: Follow [Effective Go](https://go.dev/doc/effective_go) and `golangci-lint`
- **Python**: Follow PEP 8, use `black` and `ruff`
- **TypeScript**: Follow ESLint configuration

### Pull Request Process

1. Update documentation for any new features
2. Add tests for new functionality
3. Ensure all tests pass
4. Request review from maintainers

---

## Security

### Reporting Vulnerabilities

Please report security vulnerabilities to `security@your-org.com`. Do NOT open public issues for security concerns.

### Security Measures

- JWT authentication with configurable expiry
- CSRF protection on state-changing endpoints
- Encrypted private key storage (AES-256-GCM)
- Rate limiting on sensitive endpoints
- Kubernetes secrets for sensitive configuration

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [Polymarket](https://polymarket.com) — Prediction market platform
- [NATS](https://nats.io) — High-performance messaging
- [TimescaleDB](https://www.timescale.com/) — Time-series database
- [FastAPI](https://fastapi.tiangolo.com/) — Modern Python web framework
- [Next.js](https://nextjs.org/) — React framework

---

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/your-org/polymarket-trading-bot/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/polymarket-trading-bot/discussions)

---

<div align="center">

**Built with ❤️ by the PQAP Team**

</div>
]]>