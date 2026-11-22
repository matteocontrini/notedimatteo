-- From 2026-11-21 to 2026-11-22, the editor saved bodies with the \r\n line
-- breaks that browsers send in forms. Every other body has \n line breaks.

-- +goose Up
UPDATE posts SET body = replace(body, E'\r\n', E'\n') WHERE body LIKE E'%\r\n%';
