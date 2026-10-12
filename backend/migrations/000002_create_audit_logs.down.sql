-- 000002_create_audit_logs.down.sql

DROP TRIGGER IF EXISTS audit_logs_no_update_delete ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_block_mutation();
DROP TABLE IF EXISTS audit_logs;
