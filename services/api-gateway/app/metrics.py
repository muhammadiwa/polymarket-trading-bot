from prometheus_client import Counter, Gauge, Histogram

TRADE_QUERY_LATENCY = Histogram(
    "pqap_api_trade_query_latency_ms",
    "Trade query response time in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

TRADE_QUERY_TOTAL = Counter(
    "pqap_api_trade_query_total",
    "Total trade queries served",
)

TRADE_EXPORT_TOTAL = Counter(
    "pqap_api_trade_export_total",
    "Total exports by format",
    ["format"],
)

TRADE_EXPORT_DURATION = Histogram(
    "pqap_api_trade_export_duration_ms",
    "Export duration in milliseconds",
    buckets=[100, 500, 1000, 2500, 5000, 10000],
)

TRADE_EXPORT_ROWS = Counter(
    "pqap_api_trade_export_rows_total",
    "Total rows exported",
)

PORTFOLIO_QUERY_LATENCY = Histogram(
    "pqap_api_portfolio_query_latency_ms",
    "Portfolio query response time in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

PORTFOLIO_QUERY_TOTAL = Counter(
    "pqap_api_portfolio_query_total",
    "Total portfolio queries served",
)

POSITION_QUERY_LATENCY = Histogram(
    "pqap_api_position_query_latency_ms",
    "Position query response time in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

POSITION_QUERY_TOTAL = Counter(
    "pqap_api_position_query_total",
    "Total position queries served",
)

WS_CONNECTIONS = Gauge(
    "pqap_dashboard_ws_connections_total",
    "Active dashboard WebSocket connections",
)

WS_MESSAGES_SENT = Counter(
    "pqap_dashboard_ws_messages_sent_total",
    "Total dashboard WebSocket messages sent",
)

RISK_ACTIONS_TOTAL = Counter(
    "pqap_dashboard_risk_actions_total",
    "Quick actions executed",
    ["action"],
)

RISK_ACTION_LATENCY = Histogram(
    "pqap_dashboard_risk_action_latency_ms",
    "Action execution latency in milliseconds",
    buckets=[50, 100, 250, 500, 1000, 2000],
)

RISK_PARAM_CHANGES_TOTAL = Counter(
    "pqap_dashboard_risk_param_changes_total",
    "Risk parameter adjustments",
)

HEALTH_POLL_TOTAL = Counter(
    "pqap_dashboard_health_polls_total",
    "Total health polling requests",
)

HEALTH_POLL_LATENCY = Histogram(
    "pqap_dashboard_health_poll_latency_ms",
    "Health polling latency in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

HEALTH_STALE_TOTAL = Counter(
    "pqap_dashboard_health_stale_total",
    "Stale health data served from cache",
)

OPPORTUNITY_QUERY_TOTAL = Counter(
    "pqap_dashboard_opportunities_streamed_total",
    "Total opportunity queries served",
)

OPPORTUNITY_QUERY_LATENCY = Histogram(
    "pqap_dashboard_opportunity_feed_latency_ms",
    "Opportunity query latency in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

# Admin panel metrics
ADMIN_CONFIG_CHANGES_TOTAL = Counter(
    "pqap_admin_config_changes_total",
    "Total config changes made",
)

ADMIN_CONFIG_VALIDATION_ERRORS_TOTAL = Counter(
    "pqap_admin_config_validation_errors_total",
    "Total config validation failures",
)

ADMIN_HEALTH_CHECKS_TOTAL = Counter(
    "pqap_admin_health_checks_total",
    "Total admin health check requests",
)

ADMIN_HEALTH_CHECK_LATENCY = Histogram(
    "pqap_admin_health_check_latency_ms",
    "Admin health check latency in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

ADMIN_ACTIVE_ALERTS = Gauge(
    "pqap_admin_active_alerts_total",
    "Number of active health alerts",
)

ADMIN_WS_CONNECTIONS = Gauge(
    "pqap_admin_ws_connections_total",
    "Active admin WebSocket connections",
)

# Log viewer metrics
ADMIN_LOG_QUERIES_TOTAL = Counter(
    "pqap_admin_log_queries_total",
    "Total log queries",
)

ADMIN_LOG_QUERY_LATENCY = Histogram(
    "pqap_admin_log_query_latency_ms",
    "Log query latency in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

ADMIN_LOG_INGESTION_TOTAL = Counter(
    "pqap_admin_log_ingestion_total",
    "Total log entries ingested",
)

# Database management metrics
ADMIN_BACKUP_TOTAL = Counter(
    "pqap_admin_backup_total",
    "Total backups created",
)

ADMIN_BACKUP_DURATION = Histogram(
    "pqap_admin_backup_duration_ms",
    "Backup duration in milliseconds",
    buckets=[1000, 5000, 10000, 30000, 60000, 300000, 600000],
)

ADMIN_RESTORE_TOTAL = Counter(
    "pqap_admin_restore_total",
    "Total restores performed",
)

ADMIN_CLEANUP_TOTAL = Counter(
    "pqap_admin_cleanup_total",
    "Total cleanup operations",
)

ADMIN_CLEANUP_ROWS_DELETED = Counter(
    "pqap_admin_cleanup_rows_deleted_total",
    "Total rows deleted by cleanup",
    ["table"],
)

# Cross-account metrics
PORTFOLIO_CROSS_ACCOUNT_QUERIES_TOTAL = Counter(
    "pqap_portfolio_cross_account_queries_total",
    "Total cross-account portfolio queries",
)

PORTFOLIO_CROSS_ACCOUNT_LATENCY = Histogram(
    "pqap_portfolio_cross_account_latency_ms",
    "Cross-account portfolio query latency in milliseconds",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000],
)

RISK_CROSS_ACCOUNT_EXPOSURE = Gauge(
    "pqap_risk_cross_account_exposure_total",
    "Total cross-account risk exposure",
)

RISK_PER_ACCOUNT_LIMIT_CHECKS_TOTAL = Counter(
    "pqap_risk_per_account_limit_checks_total",
    "Total per-account risk limit checks",
)

RISK_LIMITS_UPDATES_TOTAL = Counter(
    "pqap_risk_limits_updates_total",
    "Total risk limits updates",
)
