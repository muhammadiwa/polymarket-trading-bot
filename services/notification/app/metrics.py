from prometheus_client import Counter, Gauge, Histogram

NOTIFICATION_SENT_TOTAL = Counter(
    "pqap_notification_sent_total",
    "Total notifications sent successfully",
    ["channel", "severity"],
)

NOTIFICATION_DELIVERY_FAILURES_TOTAL = Counter(
    "pqap_notification_delivery_failures_total",
    "Total notifications that failed delivery",
    ["channel", "reason"],
)

NOTIFICATION_THROTTLED_TOTAL = Counter(
    "pqap_notification_throttled_total",
    "Total notifications throttled by rate limiter",
    ["severity"],
)

NOTIFICATION_SUPPRESSED_TOTAL = Counter(
    "pqap_notification_suppressed_total",
    "Total notifications suppressed by preferences",
    ["severity"],
)

NOTIFICATION_DELIVERY_LATENCY = Histogram(
    "pqap_notification_delivery_latency_ms",
    "Notification delivery latency in milliseconds",
    ["channel"],
    buckets=[100, 250, 500, 1000, 2500, 5000, 10000],
)

NOTIFICATION_QUEUE_SIZE = Gauge(
    "pqap_notification_queue_size",
    "Current number of notifications in processing queue",
)

NOTIFICATION_HISTORY_SIZE = Gauge(
    "pqap_notification_history_size",
    "Current number of notifications in history",
)
