-- Down migration for V2 schema
-- Drop tables in reverse order to handle foreign key constraints

DROP TABLE IF EXISTS competition_details;
DROP TABLE IF EXISTS ratings;
DROP TABLE IF EXISTS competition_teams;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS competitions;
DROP TABLE IF EXISTS sports; 