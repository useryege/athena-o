SELECT 'CREATE DATABASE worm_markets'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'worm_markets')\gexec

SELECT 'CREATE DATABASE wallet'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'wallet')\gexec

SELECT 'CREATE DATABASE worm_trading'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'worm_trading')\gexec

SELECT 'CREATE DATABASE sports_live'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'sports_live')\gexec

SELECT 'CREATE DATABASE sports_history'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'sports_history')\gexec

SELECT 'CREATE DATABASE managed_oo'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'managed_oo')\gexec

SELECT 'CREATE DATABASE profit_sharing'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'profit_sharing')\gexec

SELECT 'CREATE DATABASE token'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'token')\gexec

SELECT 'CREATE DATABASE temporal'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'temporal')\gexec

SELECT 'CREATE DATABASE temporal_visibility'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'temporal_visibility')\gexec
