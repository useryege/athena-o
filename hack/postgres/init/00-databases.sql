SELECT 'CREATE DATABASE application'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'application')\gexec

SELECT 'CREATE DATABASE worm'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'worm')\gexec
