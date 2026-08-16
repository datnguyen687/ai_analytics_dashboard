-- migrate:up
ALTER TABLE users RENAME COLUMN email TO username;

-- migrate:down
ALTER TABLE users RENAME COLUMN username TO email;
