#!/bin/sh
set -eu

/app/migrate
if [ "${APP_ENV:-development}" = "production" ] && [ "${SEED_DEMO_DATA:-false}" = "true" ]; then
    echo "Demo data seeding is disabled in production." >&2
    exit 1
fi
if [ "${SEED_DEMO_DATA:-false}" = "true" ]; then
    /app/seed
fi
exec /app/api
