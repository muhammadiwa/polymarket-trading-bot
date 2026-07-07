from datetime import datetime, timedelta, timezone
from decimal import Decimal

import pytest

from app.engine.overfitting_detector import detect_overfitting


def _make_trades(count: int, win_rate_in_sample: float, win_rate_out_sample: float) -> list[dict]:
    """Create test trades with different win rates for in-sample and out-of-sample."""
    split_idx = int(count * 0.7)
    trades = []
    base_time = datetime(2026, 1, 1, tzinfo=timezone.utc)

    for i in range(count):
        is_in_sample = i < split_idx
        is_win = (is_in_sample and i < split_idx * win_rate_in_sample) or (
            not is_in_sample and (i - split_idx) < (count - split_idx) * win_rate_out_sample
        )

        trades.append({
            "fill_timestamp": base_time + timedelta(hours=i),
            "pnl": Decimal("10") if is_win else Decimal("-5"),
            "score": Decimal("0.03"),
            "quantity": Decimal("100"),
        })

    return trades


class TestDetectOverfitting:
    def test_no_overfitting(self):
        """Similar win rates in-sample and out-of-sample should not flag overfitting."""
        trades = _make_trades(100, 0.70, 0.65)

        result = detect_overfitting(
            trades=trades,
            pattern_type="score_threshold",
            parameter_name="min_score",
            split_value="0.02",
        )

        assert result["is_overfitting"] is False
        assert result["degradation_pct"] is not None
        assert float(result["degradation_pct"]) < 20.0

    def test_overfitting_detected(self):
        """Large degradation in out-of-sample should flag overfitting."""
        trades = _make_trades(100, 0.80, 0.40)

        result = detect_overfitting(
            trades=trades,
            pattern_type="score_threshold",
            parameter_name="min_score",
            split_value="0.02",
        )

        assert result["is_overfitting"] is True
        assert result["degradation_pct"] is not None
        assert float(result["degradation_pct"]) > 20.0
        assert result["overfitting_score"] is not None

    def test_insufficient_data(self):
        """Too few trades should return empty result."""
        trades = _make_trades(5, 0.70, 0.65)

        result = detect_overfitting(
            trades=trades,
            pattern_type="score_threshold",
            parameter_name="min_score",
            split_value="0.02",
        )

        assert result["is_overfitting"] is False
        assert result["overfitting_score"] is None
        assert result["degradation_pct"] is None

    def test_zero_in_sample_rate(self):
        """Zero in-sample rate should return empty result (avoid division by zero)."""
        trades = []
        base_time = datetime(2026, 1, 1, tzinfo=timezone.utc)
        for i in range(100):
            trades.append({
                "fill_timestamp": base_time + timedelta(hours=i),
                "pnl": Decimal("-5"),
                "score": Decimal("0.03"),
                "quantity": Decimal("100"),
            })

        result = detect_overfitting(
            trades=trades,
            pattern_type="score_threshold",
            parameter_name="min_score",
            split_value="0.02",
        )

        assert result["is_overfitting"] is False
        assert result["overfitting_score"] is None

    def test_time_of_day_pattern(self):
        """Overfitting detection should work with time_of_day pattern."""
        trades = []
        base_time = datetime(2026, 1, 1, tzinfo=timezone.utc)
        for i in range(100):
            hour = 10 if i < 70 else (i % 24)
            trades.append({
                "fill_timestamp": base_time.replace(hour=hour) + timedelta(days=i // 24),
                "pnl": Decimal("10") if i % 3 == 0 else Decimal("-5"),
                "score": Decimal("0.03"),
                "quantity": Decimal("100"),
            })

        result = detect_overfitting(
            trades=trades,
            pattern_type="time_of_day",
            parameter_name="trading_hours",
            split_value="10:00",
        )

        assert result["is_overfitting"] is not None
        assert result["in_sample_win_rate"] is not None

    def test_position_sizing_pattern(self):
        """Overfitting detection should work with position_sizing pattern."""
        trades = []
        base_time = datetime(2026, 1, 1, tzinfo=timezone.utc)
        for i in range(100):
            trades.append({
                "fill_timestamp": base_time + timedelta(hours=i),
                "pnl": Decimal("10") if i % 2 == 0 else Decimal("-5"),
                "score": Decimal("0.03"),
                "quantity": Decimal("30") if i < 70 else Decimal("80"),
            })

        result = detect_overfitting(
            trades=trades,
            pattern_type="position_sizing",
            parameter_name="max_position_size",
            split_value="50",
        )

        assert result["is_overfitting"] is not None
        assert result["in_sample_win_rate"] is not None

    def test_unknown_pattern_type(self):
        """Unknown pattern type should use all trades (no filtering)."""
        trades = _make_trades(100, 0.70, 0.65)

        result = detect_overfitting(
            trades=trades,
            pattern_type="unknown_pattern",
            parameter_name="unknown",
            split_value="0",
        )

        assert result["in_sample_win_rate"] is not None
        assert result["out_of_sample_win_rate"] is not None
