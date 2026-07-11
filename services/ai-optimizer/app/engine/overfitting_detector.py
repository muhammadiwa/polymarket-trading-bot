import logging
from datetime import datetime
from decimal import Decimal, InvalidOperation
from typing import Optional

from scipy import stats

logger = logging.getLogger(__name__)

ZERO = Decimal("0")
OVERFITTING_THRESHOLD = 0.20
IN_SAMPLE_RATIO = 0.70
MIN_SAMPLES_PER_SPLIT = 5


def detect_overfitting(trades: list[dict], pattern_type: str, parameter_name: str, split_value: str) -> dict:
    """Detect overfitting by splitting data into in-sample and out-of-sample.

    Args:
        trades: List of trade dicts with fill_timestamp, pnl, and pattern-relevant fields
        pattern_type: Type of pattern (time_of_day, score_threshold, position_sizing)
        parameter_name: Name of parameter being evaluated
        split_value: The suggested value/threshold for the pattern

    Returns:
        Dict with overfitting_score, in_sample_win_rate, out_sample_win_rate, degradation_pct, is_overfitting
    """
    if len(trades) < MIN_SAMPLES_PER_SPLIT * 2:
        return _empty_result()

    sorted_trades = sorted(trades, key=lambda t: t.get("fill_timestamp") or t.get("created_at") or "")

    split_idx = int(len(sorted_trades) * IN_SAMPLE_RATIO)
    in_sample = sorted_trades[:split_idx]
    out_sample = sorted_trades[split_idx:]

    if len(in_sample) < MIN_SAMPLES_PER_SPLIT or len(out_sample) < MIN_SAMPLES_PER_SPLIT:
        return _empty_result()

    in_sample_rate = _calculate_win_rate(in_sample, pattern_type, parameter_name, split_value)
    out_sample_rate = _calculate_win_rate(out_sample, pattern_type, parameter_name, split_value)

    if in_sample_rate == 0:
        return _empty_result()

    degradation = (in_sample_rate - out_sample_rate) / in_sample_rate
    is_overfitting = degradation > OVERFITTING_THRESHOLD

    overfitting_score = Decimal(str(round(max(0, degradation), 4)))

    result = {
        "overfitting_score": overfitting_score,
        "in_sample_win_rate": Decimal(str(round(in_sample_rate, 4))),
        "out_of_sample_win_rate": Decimal(str(round(out_sample_rate, 4))),
        "degradation_pct": Decimal(str(round(degradation * 100, 2))),
        "is_overfitting": is_overfitting,
    }

    if is_overfitting:
        logger.warning(
            "overfitting detected",
            extra={
                "pattern_type": pattern_type,
                "in_sample_rate": in_sample_rate,
                "out_sample_rate": out_sample_rate,
                "degradation_pct": round(degradation * 100, 2),
            },
        )

    return result


def _calculate_win_rate(
    trades: list[dict],
    pattern_type: str,
    parameter_name: str,
    split_value: str,
) -> float:
    """Calculate win rate for trades matching the pattern criteria."""
    filtered = _filter_by_pattern(trades, pattern_type, parameter_name, split_value)

    if len(filtered) == 0:
        return 0.0

    wins = sum(1 for t in filtered if _safe_pnl(t) > ZERO)
    return wins / len(filtered)


def _safe_pnl(trade: dict) -> Decimal:
    """Safely extract pnl, returning 0 for None or invalid values."""
    pnl = trade.get("pnl")
    if pnl is None:
        return ZERO
    try:
        return Decimal(str(pnl))
    except (InvalidOperation, ValueError, TypeError):
        return ZERO


def _filter_by_pattern(
    trades: list[dict],
    pattern_type: str,
    parameter_name: str,
    split_value: str,
) -> list[dict]:
    """Filter trades based on pattern type and threshold."""
    if pattern_type == "time_of_day":
        try:
            target_hour = int(split_value.split(":")[0]) if ":" in split_value else int(split_value)
        except (ValueError, IndexError):
            logger.warning("invalid split_value for time_of_day", extra={"split_value": split_value})
            return []
        return [t for t in trades if _get_hour(t) == target_hour]
    elif pattern_type == "score_threshold":
        try:
            threshold = Decimal(split_value)
        except InvalidOperation:
            logger.warning("invalid split_value for score_threshold", extra={"split_value": split_value})
            return []
        return [t for t in trades if _safe_decimal(t.get("score", "0")) >= threshold]
    elif pattern_type == "position_sizing":
        try:
            threshold = Decimal(split_value)
        except InvalidOperation:
            logger.warning("invalid split_value for position_sizing", extra={"split_value": split_value})
            return []
        return [t for t in trades if _safe_decimal(t.get("quantity", "0")) < threshold]
    else:
        logger.warning("unknown pattern_type", extra={"pattern_type": pattern_type})
        return []


def _safe_decimal(value) -> Decimal:
    """Safely convert value to Decimal, returning 0 for None or invalid values."""
    if value is None:
        return ZERO
    try:
        return Decimal(str(value))
    except (InvalidOperation, ValueError, TypeError):
        return ZERO


def _get_hour(trade: dict) -> Optional[int]:
    ts = trade.get("fill_timestamp") or trade.get("created_at")
    if ts is None:
        return None
    if isinstance(ts, datetime):
        return ts.hour
    if isinstance(ts, str):
        try:
            dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
            return dt.hour
        except (ValueError, TypeError):
            return None
    return None


def _empty_result() -> dict:
    return {
        "overfitting_score": None,
        "in_sample_win_rate": None,
        "out_sample_win_rate": None,
        "degradation_pct": None,
        "is_overfitting": False,
    }
