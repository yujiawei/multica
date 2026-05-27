ALTER TABLE issue DROP COLUMN IF EXISTS stage_results;
ALTER TABLE issue DROP COLUMN IF EXISTS current_stage;
ALTER TABLE issue DROP COLUMN IF EXISTS pipeline_template_id;
DROP TABLE IF EXISTS pipeline_template;
