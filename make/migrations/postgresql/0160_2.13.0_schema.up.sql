ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS op_desc varchar(500);
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS op_result varchar(50);
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS source_ip varchar(50);
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS payload text;
