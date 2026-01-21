#!/bin/bash

# Database Migration Runner Script
# This script applies database migrations to MySQL using environment variables
# Requirements: 2.2

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if required environment variables are set
check_env_vars() {
    local missing_vars=()
    
    if [ -z "$MYSQL_HOST" ]; then
        missing_vars+=("MYSQL_HOST")
    fi
    
    if [ -z "$MYSQL_PORT" ]; then
        missing_vars+=("MYSQL_PORT")
    fi
    
    if [ -z "$MYSQL_DATABASE" ]; then
        missing_vars+=("MYSQL_DATABASE")
    fi
    
    if [ -z "$MYSQL_USER" ]; then
        missing_vars+=("MYSQL_USER")
    fi
    
    if [ -z "$MYSQL_PASSWORD" ]; then
        missing_vars+=("MYSQL_PASSWORD")
    fi
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        log_error "Missing required environment variables: ${missing_vars[*]}"
        log_info "Please set the following environment variables:"
        log_info "  MYSQL_HOST - MySQL server hostname"
        log_info "  MYSQL_PORT - MySQL server port (default: 3306)"
        log_info "  MYSQL_DATABASE - Database name"
        log_info "  MYSQL_USER - MySQL username"
        log_info "  MYSQL_PASSWORD - MySQL password"
        exit 1
    fi
}

# Check if mysql client is installed
check_mysql_client() {
    if ! command -v mysql &> /dev/null; then
        log_error "mysql client is not installed"
        log_info "Please install mysql client:"
        log_info "  macOS: brew install mysql-client"
        log_info "  Ubuntu/Debian: apt-get install mysql-client"
        log_info "  CentOS/RHEL: yum install mysql"
        exit 1
    fi
}

# Test database connection
test_connection() {
    log_info "Testing database connection..."
    
    if mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1;" &> /dev/null; then
        log_info "Database connection successful"
        return 0
    else
        log_error "Failed to connect to database"
        log_error "Host: $MYSQL_HOST:$MYSQL_PORT"
        log_error "Database: $MYSQL_DATABASE"
        log_error "User: $MYSQL_USER"
        return 1
    fi
}

# Create database if it doesn't exist
create_database() {
    log_info "Checking if database '$MYSQL_DATABASE' exists..."
    
    if mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" \
        -e "CREATE DATABASE IF NOT EXISTS \`$MYSQL_DATABASE\`;" &> /dev/null; then
        log_info "Database '$MYSQL_DATABASE' is ready"
        return 0
    else
        log_error "Failed to create database '$MYSQL_DATABASE'"
        return 1
    fi
}

# Apply migrations
apply_migrations() {
    local migrations_dir="$(dirname "$0")/../migrations"
    
    if [ ! -d "$migrations_dir" ]; then
        log_error "Migrations directory not found: $migrations_dir"
        exit 1
    fi
    
    log_info "Applying migrations from: $migrations_dir"
    
    # Find all .sql files and sort them
    local migration_files=($(find "$migrations_dir" -name "*.sql" | sort))
    
    if [ ${#migration_files[@]} -eq 0 ]; then
        log_warn "No migration files found in $migrations_dir"
        return 0
    fi
    
    log_info "Found ${#migration_files[@]} migration file(s)"
    
    # Apply each migration
    for migration_file in "${migration_files[@]}"; then
        local filename=$(basename "$migration_file")
        log_info "Applying migration: $filename"
        
        if mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" \
            "$MYSQL_DATABASE" < "$migration_file"; then
            log_info "✓ Successfully applied: $filename"
        else
            log_error "✗ Failed to apply: $filename"
            exit 1
        fi
    done
    
    log_info "All migrations applied successfully"
}

# Main execution
main() {
    log_info "Starting database migration process..."
    log_info "Target: $MYSQL_USER@$MYSQL_HOST:$MYSQL_PORT/$MYSQL_DATABASE"
    
    check_env_vars
    check_mysql_client
    test_connection
    create_database
    apply_migrations
    
    log_info "Migration process completed successfully!"
}

# Run main function
main
