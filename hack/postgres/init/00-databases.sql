SELECT 'CREATE DATABASE worm'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'worm')\gexec

SELECT 'CREATE DATABASE wormpoly'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'wormpoly')\gexec

SELECT 'CREATE DATABASE notification'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'notification')\gexec

SELECT 'CREATE DATABASE wallet'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'wallet')\gexec

SELECT 'CREATE DATABASE polymarket'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'polymarket')\gexec

SELECT 'CREATE DATABASE token'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'token')\gexec

SELECT 'CREATE DATABASE temporal'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'temporal')\gexec

SELECT 'CREATE DATABASE temporal_visibility'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'temporal_visibility')\gexec
