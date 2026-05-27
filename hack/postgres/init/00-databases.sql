SELECT 'CREATE DATABASE application'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'application')\gexec

SELECT 'CREATE DATABASE worm'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'worm')\gexec

SELECT 'CREATE DATABASE notification'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'notification')\gexec

SELECT 'CREATE DATABASE wallet'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'wallet')\gexec
