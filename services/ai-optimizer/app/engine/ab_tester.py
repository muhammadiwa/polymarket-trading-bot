import logging
import math
from decimal import Decimal
from typing import Optional

from scipy import stats

from app.models.ab_test import ABTestResultSummary, VariantStats

logger = logging.getLogger(__name__)

ZERO = Decimal("0")
MIN_SAMPLES_FOR_TEST = 10


def calculate_significance(
    control_pnls: list[Decimal],
    treatment_pnls: list[Decimal],
) -> ABTestResultSummary:
    """Calculate statistical significance between control and treatment PnL distributions."""
    if len(control_pnls) < MIN_SAMPLES_FOR_TEST or len(treatment_pnls) < MIN_SAMPLES_FOR_TEST:
        raise ValueError(
            f"Insufficient samples: control={len(control_pnls)}, treatment={len(treatment_pnls)}, "
            f"minimum={MIN_SAMPLES_FOR_TEST}"
        )

    control_floats = [float(p) for p in control_pnls]
    treatment_floats = [float(p) for p in treatment_pnls]

    control_mean = sum(control_floats) / len(control_floats)
    treatment_mean = sum(treatment_floats) / len(treatment_floats)
    mean_diff = treatment_mean - control_mean

    control_std = _stddev(control_floats, control_mean)
    treatment_std = _stddev(treatment_floats, treatment_mean)

    try:
        _, p_value = stats.ttest_ind(treatment_floats, control_floats)
    except Exception as e:
        logger.error("ttest_ind failed", extra={"error": str(e)})
        p_value = 1.0

    if p_value is None or math.isnan(p_value):
        p_value = 1.0

    is_significant = p_value < 0.05

    if is_significant and mean_diff > 0:
        recommendation = "recommend"
    elif is_significant and mean_diff < 0:
        recommendation = "reject"
    else:
        recommendation = "inconclusive"

    control_stats = VariantStats(
        count=len(control_pnls),
        mean_pnl=Decimal(str(round(control_mean, 8))),
        std_pnl=Decimal(str(round(control_std, 8))),
    )
    treatment_stats = VariantStats(
        count=len(treatment_pnls),
        mean_pnl=Decimal(str(round(treatment_mean, 8))),
        std_pnl=Decimal(str(round(treatment_std, 8))),
    )

    logger.info(
        "ab test significance calculated",
        extra={
            "control_count": len(control_pnls),
            "treatment_count": len(treatment_pnls),
            "p_value": p_value,
            "mean_diff": mean_diff,
            "recommendation": recommendation,
        },
    )

    return ABTestResultSummary(
        ab_test_id="",
        control=control_stats,
        treatment=treatment_stats,
        p_value=round(p_value, 8),
        mean_difference=Decimal(str(round(mean_diff, 8))),
        is_significant=is_significant,
        recommendation=recommendation,
    )


def simulate_trade_outcome(
    entry_price: Decimal,
    exit_price: Optional[Decimal],
    quantity: Decimal,
    side: str,
) -> Optional[Decimal]:
    """Calculate PnL for a simulated trade."""
    if exit_price is None:
        return None

    if side == "YES":
        pnl = (exit_price - entry_price) * quantity
    else:
        pnl = (entry_price - exit_price) * quantity

    return pnl


def _stddev(values: list[float], mean: float) -> float:
    if len(values) < 2:
        return 0.0
    variance = sum((x - mean) ** 2 for x in values) / (len(values) - 1)
    return math.sqrt(variance)
