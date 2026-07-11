import Decimal from "decimal.js";

export function formatCurrency(value: string, decimals = 2): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN()) return "$0.00";
    const sign = num.isNeg() ? "-" : "";
    return `${sign}$${num.abs().toFixed(decimals)}`;
  } catch {
    return "$0.00";
  }
}

export function pnlColor(value: string): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN() || num.isZero()) return "text-gray-400";
    return num.isPos() ? "text-[#00ff88]" : "text-[#ff4757]";
  } catch {
    return "text-gray-400";
  }
}
