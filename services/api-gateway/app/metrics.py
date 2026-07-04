from prometheus_client import Counter, Histogram

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
