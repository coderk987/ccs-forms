BEGIN;

-- USERS
INSERT INTO users (gmail, name, role) VALUES
    ('alice@example.com', 'Alice Sharma', 'admin'),
    ('bob@example.com', 'Bob Singh', 'user'),
    ('charlie@example.com', 'Charlie Kumar', 'user');

COMMIT;