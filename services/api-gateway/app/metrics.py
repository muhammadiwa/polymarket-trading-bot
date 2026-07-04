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
