"use client";

import { useState, useCallback, useEffect } from "react";
import { updateRiskParameters } from "@/lib/api";
import type { RiskParameterUpdate } from "@/types";
import Decimal from "decimal.js";

interface RiskParamFormProps {
  currentValues?: {
    dailyLossLimit?: string;
    maxPositionPerMarket?: string;
    maxPositionPerStrategy?: string;
  };
  onSuccess?: () => void;
}

const SAFE_RANGES: Record<string, { min: number; max: number; label: string }> = {
  dailyLossLimit: { min: 1, max: 20, label: "Daily Loss Limit (%)" },
  maxPositionPerMarket: { min: 1, max: 50, label: "Max Position per Market (%)" },
  maxPositionPerStrategy: { min: 1, max: 50, label: "Max Position per Strategy (%)" },
};

function validatePercentage(value: string, field: string): string | null {
  const range = SAFE_RANGES[field];
  if (!range) return "Unknown field";

  try {
    const num = new Decimal(value);
    if (num.isNaN() || num.isNeg()) return "Must be a positive number";
    if (num.lt(range.min) || num.gt(range.max)) return `Must be between ${range.min}% and ${range.max}%`;
    return null;
  } catch {
    return "Invalid number";
  }
}

export function RiskParamForm({ currentValues, onSuccess }: RiskParamFormProps) {
  const [values, setValues] = useState<RiskParameterUpdate>({
    dailyLossLimit: currentValues?.dailyLossLimit ?? "",
    maxPositionPerMarket: currentValues?.maxPositionPerMarket ?? "",
    maxPositionPerStrategy: currentValues?.maxPositionPerStrategy ?? "",
  });
  const [errors, setErrors] = useState<Record<string, string | null>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState(false);

  // #8: Sync values when currentValues prop changes
  useEffect(() => {
    if (currentValues) {
      setValues({
        dailyLossLimit: currentValues.dailyLossLimit ?? "",
        maxPositionPerMarket: currentValues.maxPositionPerMarket ?? "",
        maxPositionPerStrategy: currentValues.maxPositionPerStrategy ?? "",
      });
    }
  }, [currentValues?.dailyLossLimit, currentValues?.maxPositionPerMarket, currentValues?.maxPositionPerStrategy]);

  const handleChange = useCallback((field: keyof RiskParameterUpdate, value: string) => {
    setValues((prev) => ({ ...prev, [field]: value }));
    setErrors((prev) => ({ ...prev, [field]: null }));
    setSubmitSuccess(false);
    setSubmitError(null);
  }, []);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();

    const newErrors: Record<string, string | null> = {};
    const params: RiskParameterUpdate = {};

    for (const [field, value] of Object.entries(values)) {
      if (!value || value.trim() === "") continue;
      const err = validatePercentage(value, field);
      if (err) {
        newErrors[field] = err;
      } else {
        params[field as keyof RiskParameterUpdate] = value;
      }
    }

    setErrors(newErrors);
    if (Object.values(newErrors).some((e) => e !== null)) return;
    if (Object.keys(params).length === 0) {
      setSubmitError("No changes to save");
      return;
    }

    setSubmitting(true);
    setSubmitError(null);
    try {
      await updateRiskParameters(params);
      setSubmitSuccess(true);
      onSuccess?.();
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Failed to update parameters");
    } finally {
      setSubmitting(false);
    }
  }, [values, onSuccess]);

  return (
    <form onSubmit={handleSubmit} className="space-y-4" aria-label="Risk Parameter Adjustment">
      {(Object.entries(SAFE_RANGES) as [keyof RiskParameterUpdate, { min: number; max: number; label: string }][]).map(([field, range]) => (
        <div key={field}>
          <label htmlFor={`risk-${field}`} className="block text-sm text-gray-400 mb-1">
            {range.label}
          </label>
          <div className="flex items-center gap-2">
            <input
              id={`risk-${field}`}
              type="text"
              inputMode="decimal"
              placeholder={`${range.min} - ${range.max}`}
              value={values[field] ?? ""}
              onChange={(e) => handleChange(field, e.target.value)}
              className={`flex-1 px-3 py-2 rounded-lg border bg-white/5 text-white font-mono text-sm placeholder-gray-500 focus:outline-none focus:ring-2 transition-colors ${
                errors[field]
                  ? "border-[#ff4757]/50 focus:ring-[#ff4757]/30"
                  : "border-white/10 focus:ring-[#00d4ff]/30"
              }`}
              aria-invalid={!!errors[field]}
              aria-describedby={errors[field] ? `error-${field}` : undefined}
            />
            <span className="text-sm text-gray-400">%</span>
          </div>
          {errors[field] && (
            <p id={`error-${field}`} className="text-xs text-[#ff4757] mt-1" role="alert">
              {errors[field]}
            </p>
          )}
        </div>
      ))}

      {submitError && (
        <div className="rounded-lg border border-[#ff4757]/30 bg-[#ff4757]/10 p-3 text-sm text-[#ff4757]" role="alert">
          {submitError}
        </div>
      )}

      {submitSuccess && (
        <div className="rounded-lg border border-[#00ff88]/30 bg-[#00ff88]/10 p-3 text-sm text-[#00ff88]" role="status">
          Parameters updated successfully
        </div>
      )}

      <button
        type="submit"
        disabled={submitting}
        className="w-full px-4 py-2 rounded-lg bg-[#00d4ff] text-black font-medium hover:bg-[#00d4ff]/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      >
        {submitting ? "Saving..." : "Save Parameters"}
      </button>
    </form>
  );
}
