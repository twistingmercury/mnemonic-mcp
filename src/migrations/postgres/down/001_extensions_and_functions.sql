-- src/migrations/postgres/down/001_extensions_and_functions.sql
-- Reverses: Enables required PostgreSQL extensions.
-- WARNING: Dropping extensions may fail if objects depend on them.

-- Extensions are not dropped to avoid breaking other schemas that may use them.
-- If you need to drop extensions, uncomment the following lines:
-- drop extension if exists vector;
-- drop extension if exists "uuid-ossp";
