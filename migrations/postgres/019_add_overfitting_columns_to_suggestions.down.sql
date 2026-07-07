ALTER TABLE optimizer_suggestions
DROP COLUMN IF EXISTS overfitting_score,
DROP COLUMN IF EXISTS out_of_sample_win_rate,
DROP COLUMN IF EXISTS in_sample_win_rate,
DROP COLUMN IF EXISTS degradation_pct;
