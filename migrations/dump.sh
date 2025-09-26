#!/bin/sh

docker exec -i akood-dev-postgres_db-1 pg_dump -U solo -d akood-mvp --schema-only --no-owner --no-privileges
