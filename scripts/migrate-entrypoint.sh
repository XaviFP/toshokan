#!/bin/sh
set -e

if [ -z "$SERVICE" ]; then
    echo "ERROR: SERVICE environment variable is required (deck, user, course, or all)"
    exit 1
fi

echo "Running migrations for service: $SERVICE"

case "$SERVICE" in
    deck)
        DB_NAME=${DB_NAME:-deck} MIGRATIONS_PATH="file:///migrations/deck" /migrate-deck
        ;;
    user)
        DB_NAME=${DB_NAME:-users} MIGRATIONS_PATH="file:///migrations/user" /migrate-user
        ;;
    course)
        DB_NAME=${DB_NAME:-course} MIGRATIONS_PATH="file:///migrations/course" /migrate-course
        ;;
    all)
        echo "Running all migrations..."
        DB_NAME=deck MIGRATIONS_PATH="file:///migrations/deck" /migrate-deck
        DB_NAME=users MIGRATIONS_PATH="file:///migrations/user" /migrate-user
        DB_NAME=course MIGRATIONS_PATH="file:///migrations/course" /migrate-course
        echo "All migrations completed."
        ;;
    *)
        echo "ERROR: Unknown service '$SERVICE'. Valid options: deck, user, course, all"
        exit 1
        ;;
esac
