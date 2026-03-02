-- Rollback migration - reverse everything from 001_init.up.sql
-- Reference: https://www.postgresql.org/docs/current/sql-droptable.html
-- CASCADE option: automatically drops dependent objects

DROP TABLE IF EXISTS japanese_dictionary CASCADE;
DROP TABLE IF EXISTS analysis_tokens CASCADE;
DROP TABLE IF EXISTS analyses CASCADE;
DROP TABLE IF EXISTS known_words CASCADE;
DROP TABLE IF EXISTS users CASCADE;

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();