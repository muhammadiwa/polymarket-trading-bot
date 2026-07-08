ALTER TABLE optimizer_suggestions
ADD COLUMN overfitting_score DECIMAL(5,4),
ADD COLUMN out_of_sample_win_rate DECIMAL(5,4),
ADD COLUMN in_sample_win_rate DECIMAL(5,4),
ADD COLUMN degradation_pct DECIMAL(5,2);
