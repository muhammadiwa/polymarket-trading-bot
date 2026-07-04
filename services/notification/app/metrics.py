from prometheus_client import Counter, Gauge, Histogram

NOTIFICATION_SENT_TOTAL = Counter(
    "pqap_notification_sent_total",
    "Total notifications sent successfully",
    ["channel", "severity"],
)

NOTIFICATION_FAILED_TOTAL = Counter(
    "pqap_notification_failed_total",
    "Total notifications that failed delivery",
    ["channel", "reason"],
)

NOTIFICATION_THROTTLED_TOTAL = Counter(
    "pqap_notification_throttled_total",
    "Total notifications throttled by rate limiter",
    ["severity"],
)

NOTIFICATION_DELIVERY_LATENCY = Histogram(
    "pqap_notification_delivery_latency_seconds",
    "Notification delivery latency in seconds",
    ["channel"],
    buckets=[0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0],
)

NOTIFICATION_QUEUE_SIZE = Gauge(
    "pqap_notification_queue_size",
    "Current number of notifications in processing queue",
)
