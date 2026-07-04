from app.models import Severity

EVENT_SEVERITY_MAP: dict[str, Severity] = {
    "EmergencyStop": Severity.CRITICAL,
    "CircuitBreakerTripped": Severity.CRITICAL,
    "APIFailure": Severity.CRITICAL,
    "DrawdownBreach": Severity.CRITICAL,
    "DailyBudgetExhausted": Severity.CRITICAL,
    "DailyBudget80Percent": Severity.WARNING,
    "DrawdownApproaching": Severity.WARNING,
    "PositionLimitBreach": Severity.WARNING,
    "WinStreak": Severity.WARNING,
    "OrderFilled": Severity.INFO,
    "TradeExecuted": Severity.INFO,
    "StrategyOptimization": Severity.INFO,
    "SystemHealth": Severity.DEBUG,
    "ReconnectionEvent": Severity.DEBUG,
    "ReconciliationComplete": Severity.DEBUG,
}

SEVERITY_EMOJI: dict[Severity, str] = {
    Severity.CRITICAL: "\U0001f534",
    Severity.WARNING: "\U0001f7e1",
    Severity.INFO: "\U0001f535",
    Severity.DEBUG: "\u26aa",
}


def classify(event_type: str, payload_severity: Severity | None = None, priority: str | None = None) -> Severity:
    if payload_severity is not None:
        return payload_severity
    if priority:
        priority_map = {
            "critical": Severity.CRITICAL,
            "high": Severity.CRITICAL,
            "warning": Severity.WARNING,
            "medium": Severity.WARNING,
            "low": Severity.INFO,
            "info": Severity.INFO,
        }
        mapped = priority_map.get(priority.lower())
        if mapped is not None:
            return mapped
    return EVENT_SEVERITY_MAP.get(event_type, Severity.INFO)


def get_emoji(severity: Severity) -> str:
    return SEVERITY_EMOJI.get(severity, "\u26aa")
