-- Down migration for NFL sport insertion
-- Remove the NFL sport record

DELETE FROM sports WHERE id = 'nfl'; 