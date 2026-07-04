from app.categorizer import get_emoji
from app.models import NotificationRecord, Severity


def format_message(record: NotificationRecord) -> str:
    emoji = get_emoji(record.severity)
    ts = record.created_at.strftime("%Y-%m-%d %H:%M:%S UTC")

    if record.severity == Severity.CRITICAL:
        return (
            f"{emoji} CRITICAL: {record.title.upper()}\n"
            f"\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501"
            f"\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\n"
            f"{record.message}\n\n"
            f"\u23f0 {ts}\n"
            f"\U0001f4cb Event: {record.event_type}"
        )

    if record.severity == Severity.WARNING:
        return (
            f"{emoji} Warning: {record.title}\n"
            f"{record.message}\n\n"
            f"\u23f0 {ts}"
        )

    if record.severity == Severity.INFO:
        return (
            f"{emoji} Info: {record.title}\n"
            f"{record.message}\n\n"
            f"\u23f0 {ts}"
        )

    return f"{emoji} Debug: {record.title}\n{record.message}\n\n\u23f0 {ts}"
