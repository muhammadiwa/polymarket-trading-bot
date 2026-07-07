import logging
import math
from collections import defaultdict
from decimal import Decimal
from typing import Optional

from scipy import stats

logger = logging.getLogger(__name__)

ZERO = Decimal("0")

# #11: Configurable thresholds for pattern discovery
SCORE_SPLIT_THRESHOLD = Decimal("0.02")
SIZE_SPLIT_THRESHOLD = Decimal("50")
MIN_SAMPLES_PER_BUCKET = 5
MIN_TOTAL_TRADES = 10
MIN_RATE_DELTA = 0.10  # #7: Minimum win rate delta to avoid spurious patterns


async def analyze_trades(trades: list[dict]) -> list[dict]:
    """Analyze trades and return patterns with statistical significance."""
    patterns = []

    # Filter filled trades only
    filled = [t for t in trades if t.get("fill_status") in ("FILLED", "PARTIAL_FILL")]
    if len(filled) < MIN_TOTAL_TRADES:
        return patterns

    wins = [t for t in filled if Decimal(str(t.get("pnl", "0"))) > ZERO]
    losses = [t for t in filled if Decimal(str(t.get("pnl", "0"))) < ZERO]

    # Pattern 1: Time-of-day
    time_pattern = _analyze_time_of_day(filled, wins, losses)
    if time_pattern:
        patterns.append(time_pattern)

    # Pattern 2: Score threshold
    score_pattern = _analyze_score_threshold(filled, wins, losses)
    if score_pattern:
        patterns.append(score_pattern)

    # Pattern 3: Position sizing
    size_pattern = _analyze_position_sizing(filled, wins, losses)
    if size_pattern:
        patterns.append(size_pattern)

    return patterns


def _analyze_time_of_day(trades: list, wins: list, losses: list) -> Optional[dict]:
    """Analyze win rate by time of day."""
    # Group by hour
    hour_wins: dict[int, int] = defaultdict(int)
    hour_total: dict[int, int] = defaultdict(int)

    for t in trades:
        ts = t.get("fill_timestamp") or t.get("created_at")
        # #5: Skip trades with missing timestamps
        if ts and hasattr(ts, "hour"):
            hour = ts.hour
        else:
            continue
        hour_total[hour] = hour_total.get(hour, 0) + 1
        if Decimal(str(t.get("pnl", "0"))) > ZERO:
            hour_wins[hour] = hour_wins.get(hour, 0) + 1

    if len(hour_total) < MIN_SAMPLES_PER_BUCKET:
        return None

    # Find best and worst hours
    best_hour = max(hour_total.keys(), key=lambda h: hour_wins.get(h, 0) / hour_total[h] if hour_total[h] > 0 else 0)
    worst_hour = min(hour_total.keys(), key=lambda h: hour_wins.get(h, 0) / hour_total[h] if hour_total[h] > 0 else 1)

    best_rate = hour_wins.get(best_hour, 0) / hour_total[best_hour] if hour_total[best_hour] > 0 else 0
    worst_rate = hour_wins.get(worst_hour, 0) / hour_total[worst_hour] if hour_total[worst_hour] > 0 else 0

    if best_rate <= worst_rate:
        return None

    # #7: Minimum delta threshold to avoid spurious patterns from zero-PnL trades
    if best_rate - worst_rate < MIN_RATE_DELTA:
        return None

    # Chi-squared test
    observed = [hour_wins.get(best_hour, 0), hour_total[best_hour] - hour_wins.get(best_hour, 0)]
    expected_rate = len(wins) / len(trades) if trades else 0
    expected = [hour_total[best_hour] * expected_rate, hour_total[best_hour] * (1 - expected_rate)]

    if expected[0] < 5 or expected[1] < 5:
        return None

    try:
        _, p_value = stats.chisquare(observed, expected)
    except Exception:
        return None

    # #9: Guard against NaN p-values
    if p_value is None or math.isnan(p_value) or p_value >= 0.05:
        return None

    impact_pct = ((best_rate - worst_rate) / worst_rate * 100) if worst_rate > 0 else 0

    return {
        "pattern_type": "time_of_day",
        "parameter_name": "trading_hours",
        "current_value": "00:00-23:59",
        "suggested_value": f"{best_hour:02d}:00-{(best_hour+12)%24:02d}:00",
        "expected_impact": f"+{impact_pct:.0f}% win rate during {best_hour:02d}:00-{(best_hour+12)%24:02d}:00 UTC",
        "methodology": f"Trades at hour {best_hour} UTC show {best_rate*100:.0f}% win rate vs {worst_rate*100:.0f}% at hour {worst_hour} (p={p_value:.4f}, n={len(trades)}). Restricting to profitable hours eliminates losing trades outside the window.",
        "confidence": min(0.95, 1 - p_value),
        "p_value": p_value,
    }


def _analyze_score_threshold(trades: list, wins: list, losses: list) -> Optional[dict]:
    """Analyze win rate by score threshold."""
    # #11: Use configurable threshold for score split
    low_score = [t for t in trades if Decimal(str(t.get("score", "0"))) < SCORE_SPLIT_THRESHOLD]
    high_score = [t for t in trades if Decimal(str(t.get("score", "0"))) >= SCORE_SPLIT_THRESHOLD]

    if len(low_score) < MIN_SAMPLES_PER_BUCKET or len(high_score) < MIN_SAMPLES_PER_BUCKET:
        return None

    low_wins = sum(1 for t in low_score if Decimal(str(t.get("pnl", "0"))) > ZERO)
    high_wins = sum(1 for t in high_score if Decimal(str(t.get("pnl", "0"))) > ZERO)

    low_rate = low_wins / len(low_score)
    high_rate = high_wins / len(high_score)

    if high_rate <= low_rate:
        return None

    # T-test on PnL values
    low_pnls = [float(t.get("pnl", 0)) for t in low_score]
    high_pnls = [float(t.get("pnl", 0)) for t in high_score]

    try:
        _, p_value = stats.ttest_ind(high_pnls, low_pnls)
    except Exception:
        return None

    if p_value is None or math.isnan(p_value) or p_value >= 0.05:
        return None

    impact_pct = ((high_rate - low_rate) / low_rate * 100) if low_rate > 0 else 0

    return {
        "pattern_type": "score_threshold",
        "parameter_name": "min_score",
        "current_value": "0.01",
        "suggested_value": "0.02",
        "expected_impact": f"+{impact_pct:.0f}% win rate with higher score threshold",
        "methodology": f"Trades with score >= 0.02 show {high_rate*100:.0f}% win rate vs {low_rate*100:.0f}% below (p={p_value:.4f}, n={len(trades)}). Raising min_score filters low-quality trades.",
        "confidence": min(0.95, 1 - p_value),
        "p_value": p_value,
    }


def _analyze_position_sizing(trades: list, wins: list, losses: list) -> Optional[dict]:
    """Analyze PnL by position size."""
    # #11: Use configurable threshold for size split
    small = [t for t in trades if Decimal(str(t.get("quantity", "0"))) < SIZE_SPLIT_THRESHOLD]
    large = [t for t in trades if Decimal(str(t.get("quantity", "0"))) >= SIZE_SPLIT_THRESHOLD]

    if len(small) < MIN_SAMPLES_PER_BUCKET or len(large) < MIN_SAMPLES_PER_BUCKET:
        return None

    small_pnl = sum(Decimal(str(t.get("pnl", "0"))) for t in small) / len(small)
    large_pnl = sum(Decimal(str(t.get("pnl", "0"))) for t in large) / len(large)

    if small_pnl <= large_pnl:
        return None

    small_vals = [float(t.get("pnl", 0)) for t in small]
    large_vals = [float(t.get("pnl", 0)) for t in large]

    try:
        _, p_value = stats.ttest_ind(small_vals, large_vals)
    except Exception:
        return None

    if p_value is None or math.isnan(p_value) or p_value >= 0.05:
        return None

    return {
        "pattern_type": "position_sizing",
        "parameter_name": "max_position_size",
        "current_value": "100",
        "suggested_value": "50",
        "expected_impact": f"+${float(small_pnl - large_pnl):.2f} avg PnL per trade with smaller positions",
        "methodology": f"Smaller positions (< 50) average ${float(small_pnl):.2f} PnL vs ${float(large_pnl):.2f} for larger positions (p={p_value:.4f}, n={len(trades)}). Reducing position size improves risk-adjusted returns.",
        "confidence": min(0.95, 1 - p_value),
        "p_value": p_value,
    }
