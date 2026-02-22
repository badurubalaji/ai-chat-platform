#!/bin/bash

# Configuration
DB_HOST="localhost"
DB_PORT="5432"
DB_SUPERUSER="postgres" # User with CREATE ROLE/DB privileges

# App Config
APP_USER="ai_user"
APP_PASS="ai_password"
APP_DB="ai_chat_db"

echo "This script will create the user '$APP_USER' and database '$APP_DB'."
echo "It requires 'psql' to be installed and accessible."
read -p "Enter superuser ($DB_SUPERUSER) password (leave empty if none/peer auth): " PGPASSWORD

export PGPASSWORD

echo "Creating user..."
psql -h $DB_HOST -p $DB_PORT -U $DB_SUPERUSER -c "CREATE USER $APP_USER WITH PASSWORD '$APP_PASS';" || echo "User might already exist"

echo "Creating database..."
psql -h $DB_HOST -p $DB_PORT -U $DB_SUPERUSER -c "CREATE DATABASE $APP_DB OWNER $APP_USER;" || echo "Database might already exist"

echo "Granting privileges..."
psql -h $DB_HOST -p $DB_PORT -U $DB_SUPERUSER -c "GRANT ALL PRIVILEGES ON DATABASE $APP_DB TO $APP_USER;"

echo "Done! You can now run:"
echo "  ./bin/migrate up"
echo "  ./start.sh"
