#!/bin/bash
# Initialize multiple databases and users in shared MySQL instance

set -e

echo "Creating databases and users..."

# Create shortener database and user
mysql -u root -p"${MYSQL_ROOT_PASSWORD}" <<-EOSQL
    CREATE DATABASE IF NOT EXISTS shortener;
    CREATE USER IF NOT EXISTS 'shortener_user'@'%' IDENTIFIED BY 'shortener_password';
    GRANT ALL PRIVILEGES ON shortener.* TO 'shortener_user'@'%';
    
    CREATE DATABASE IF NOT EXISTS im_chat;
    CREATE USER IF NOT EXISTS 'im_service'@'%' IDENTIFIED BY 'im_service_password';
    GRANT ALL PRIVILEGES ON im_chat.* TO 'im_service'@'%';
    
    FLUSH PRIVILEGES;
EOSQL

echo "Databases and users created successfully!"

# Run shortener migrations if they exist
if [ -d "/docker-entrypoint-initdb.d/shortener" ]; then
    echo "Running shortener migrations..."
    for f in /docker-entrypoint-initdb.d/shortener/*.sql; do
        if [ -f "$f" ]; then
            echo "Executing $f..."
            mysql -u shortener_user -pshortener_password shortener < "$f"
        fi
    done
fi

echo "MySQL initialization complete!"
