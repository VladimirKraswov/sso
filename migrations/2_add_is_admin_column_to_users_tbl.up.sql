-- Добавляем в таблицу users колонку is_admin
ALTER TABLE users
    ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE;