#!/bin/bash

# ==============================================================================
# Security & Environment Initialization
# ==============================================================================
# Find the absolute path of the directory where this script lives
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Assume the .env file lives one level up in the project root
ENV_FILE="$SCRIPT_DIR/../.env"

echo "========================================================================"
echo "Disaster Recovery: Commencing Quest Log backup routine..."
echo "Timestamp: $(date)"
echo "========================================================================"

# Securely load the environment file if it exists
if [ -f "$ENV_FILE" ]; then
    echo "Security: Loading local configuration from .env..."
    # Read line by line, ignore comments, and export valid variables
    while IFS= read -r line || [ -n "$line" ]; do
        # Skip lines that are empty or start with a comment symbol
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line//[[:space:]]/}" ]] && continue
        
        # Export the variable to the current shell session
        export "$line"
    done < "$ENV_FILE"
else
    echo "Security Warning: No .env file detected at $ENV_FILE"
    echo "Falling back to system environment variables..."
fi

# ==============================================================================
# Configuration Enforcement (Fail-Closed Security)
# ==============================================================================
# If DB_PATH or BACKUP_DIR aren't provided by .env or the OS, abort immediately.
# This prevents the script from creating folders or reading files in unintended directories.
if [ -z "$DB_PATH" ] || [ -z "$BACKUP_DIR" ]; then
    echo "CRITICAL ERROR: Environment variables 'DB_PATH' or 'BACKUP_DIR' are missing." >&2
    echo "Asset protection abort triggered. Backup cancelled." >&2
    exit 1
fi

RETENTION_DAYS=${RETENTION_DAYS:-7}
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_NAME="forge_backup_$TIMESTAMP.tar.gz"
TEMP_SNAPSHOT="$BACKUP_DIR/temp_quests.db"

# Ensure the safe backup directory exists
mkdir -p "$BACKUP_DIR"

# ==============================================================================
# Phase 1: Safe SQLite Snapshot
# ==============================================================================
echo "Storage Maintenance: Creating safe SQLite snapshot..."
sqlite3 "$DB_PATH" ".backup '$TEMP_SNAPSHOT'"

if [ $? -ne 0 ]; then
    echo "CRITICAL ERROR: Failed to create SQLite live snapshot." >&2
    exit 1
fi

# ==============================================================================
# Phase 2: Compression & Telemetry
# ==============================================================================
echo "Storage Telemetry: Compacting snapshot into archive..."
tar -czf "$BACKUP_DIR/$BACKUP_NAME" -C "$BACKUP_DIR" temp_quests.db

if [ $? -eq 0 ]; then
    PRE_SIZE=$(stat -c%s "$DB_PATH" 2>/dev/null || stat -f%z "$DB_PATH")
    POST_SIZE=$(stat -c%s "$BACKUP_DIR/$BACKUP_NAME" 2>/dev/null || stat -f%z "$BACKUP_DIR/$BACKUP_NAME")
    
    echo "Storage Maintenance: Backup archived successfully: $BACKUP_NAME"
    echo "Storage Telemetry: Live DB size: $PRE_SIZE bytes | Compressed Backup size: $POST_SIZE bytes"
else
    echo "CRITICAL ERROR: Compression phase failed." >&2
    rm -f "$TEMP_SNAPSHOT"
    exit 1
fi

# Clean up the uncompressed temporary snapshot
rm -f "$TEMP_SNAPSHOT"

# ==============================================================================
# Phase 3: Rolling Window Retention
# ==============================================================================
echo "Storage Maintenance: Enforcing $RETENTION_DAYS-day backup rolling window..."

# Securely find and destroy files matching ONLY the strict naming pattern inside the target directory
find "$BACKUP_DIR" -maxdepth 1 -type f -name "forge_backup_*.tar.gz" -mtime +$RETENTION_DAYS -exec rm {} \;

echo "========================================================================"
echo "Disaster Recovery: Routine complete. System secure."
echo "========================================================================"