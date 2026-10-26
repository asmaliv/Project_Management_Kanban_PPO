-- 000005_create_risks.down.sql

DROP TABLE IF EXISTS risks;
DROP SEQUENCE IF EXISTS risk_code_seq;
DROP TYPE IF EXISTS risk_status;
DROP TYPE IF EXISTS risk_level;
DROP TYPE IF EXISTS risk_category;
