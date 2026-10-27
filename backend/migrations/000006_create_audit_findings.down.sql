-- 000006_create_audit_findings.down.sql

DROP TABLE IF EXISTS audit_findings;
DROP SEQUENCE IF EXISTS audit_finding_code_seq;
DROP TYPE IF EXISTS finding_status;
DROP TYPE IF EXISTS finding_severity;
