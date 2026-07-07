from datetime import datetime, timezone
from decimal import Decimal

import pytest

from app.engine.ab_tester import calculate_significance, simulate_trade_outcome


class TestCalculateSignificance:
    def test_significant_treatment_wins(self):
        """Treatment outperforms control with p < 0.05."""
        control = [Decimal("10")] * 20 + [Decimal("-5")] * 10
        treatment = [Decimal("20")] * 25 + [Decimal("-3")] * 5

        result = calculate_significance(control, treatment)

        assert result.is_significant is True
        assert result.recommendation == "recommend"
        assert result.p_value < 0.05
        assert result.treatment.mean_pnl > result.control.mean_pnl

    def test_significant_control_wins(self):
        """Control outperforms treatment with p < 0.05."""
        control = [Decimal("20")] * 25 + [Decimal("-3")] * 5
        treatment = [Decimal("10")] * 20 + [Decimal("-5")] * 10

        result = calculate_significance(control, treatment)

        assert result.is_significant is True
        assert result.recommendation == "reject"
        assert result.p_value < 0.05

    def test_inconclusive(self):
        """No significant difference between control and treatment."""
        control = [Decimal("10")] * 15 + [Decimal("-5")] * 15
        treatment = [Decimal("11")] * 15 + [Decimal("-4")] * 15

        result = calculate_significance(control, treatment)

        assert result.is_significant is False
        assert result.recommendation == "inconclusive"

    def test_insufficient_samples_control(self):
        """Should raise ValueError when control has too few samples."""
        control = [Decimal("10")] * 5
        treatment = [Decimal("10")] * 20

        with pytest.raises(ValueError, match="Insufficient samples"):
            calculate_significance(control, treatment)

    def test_insufficient_samples_treatment(self):
        """Should raise ValueError when treatment has too few samples."""
        control = [Decimal("10")] * 20
        treatment = [Decimal("10")] * 5

        with pytest.raises(ValueError, match="Insufficient samples"):
            calculate_significance(control, treatment)

    def test_variant_stats_correct(self):
        """Verify variant stats are calculated correctly."""
        control = [Decimal("10"), Decimal("20"), Decimal("30")] * 5  # 15 samples
        treatment = [Decimal("40"), Decimal("50"), Decimal("60")] * 5  # 15 samples

        result = calculate_significance(control, treatment)

        assert result.control.count == 15
        assert result.control.mean_pnl == Decimal("20")
        assert result.treatment.count == 15
        assert result.treatment.mean_pnl == Decimal("50")


class TestSimulateTradeOutcome:
    def test_yes_trade_profit(self):
        pnl = simulate_trade_outcome(
            entry_price=Decimal("0.60"),
            exit_price=Decimal("0.80"),
            quantity=Decimal("100"),
            side="YES",
        )
        assert pnl == Decimal("20")

    def test_yes_trade_loss(self):
        pnl = simulate_trade_outcome(
            entry_price=Decimal("0.60"),
            exit_price=Decimal("0.40"),
            quantity=Decimal("100"),
            side="YES",
        )
        assert pnl == Decimal("-20")

    def test_no_trade_profit(self):
        pnl = simulate_trade_outcome(
            entry_price=Decimal("0.40"),
            exit_price=Decimal("0.20"),
            quantity=Decimal("100"),
            side="NO",
        )
        assert pnl == Decimal("20")

    def test_no_trade_loss(self):
        pnl = simulate_trade_outcome(
            entry_price=Decimal("0.40"),
            exit_price=Decimal("0.60"),
            quantity=Decimal("100"),
            side="NO",
        )
        assert pnl == Decimal("-20")

    def test_no_exit_price(self):
        pnl = simulate_trade_outcome(
            entry_price=Decimal("0.60"),
            exit_price=None,
            quantity=Decimal("100"),
            side="YES",
        )
        assert pnl is None
