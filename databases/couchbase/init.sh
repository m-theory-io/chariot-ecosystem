#!/bin/bash

# Couchbase initialization script
set -e

echo "Starting Couchbase initialization..."

# Wait for Couchbase to start
sleep 30

# Check if cluster is already initialized
if couchbase-cli server-list -c localhost:8091 -u "$CB_ADMIN_USERNAME" -p "$CB_ADMIN_PASSWORD" >/dev/null 2>&1; then
    echo "Couchbase cluster already initialized"
    exit 0
fi

echo "Initializing Couchbase cluster..."

# Initialize the cluster
couchbase-cli cluster-init \
    --cluster localhost:8091 \
    --cluster-username "$CB_ADMIN_USERNAME" \
    --cluster-password "$CB_ADMIN_PASSWORD" \
    --cluster-name "ChariotCluster" \
    --cluster-ramsize 1024 \
    --cluster-index-ramsize 256 \
    --cluster-fts-ramsize 256 \
    --cluster-eventing-ramsize 256 \
    --cluster-analytics-ramsize 1024 \
    --services data,index,query,fts

# Wait for cluster to be ready
sleep 10

# Create bucket
echo "Creating bucket: $CB_BUCKET_NAME"
couchbase-cli bucket-create \
    --cluster localhost:8091 \
    --username "$CB_ADMIN_USERNAME" \
    --password "$CB_ADMIN_PASSWORD" \
    --bucket "$CB_BUCKET_NAME" \
    --bucket-type couchbase \
    --bucket-ramsize "$CB_BUCKET_RAM"

# Wait for bucket to be ready
sleep 5

# Create a user for the application
echo "Creating application user..."
couchbase-cli user-manage \
    --cluster localhost:8091 \
    --username "$CB_ADMIN_USERNAME" \
    --password "$CB_ADMIN_PASSWORD" \
    --set \
    --rbac-username chariot \
    --rbac-password chariot123 \
    --roles bucket_full_access["$CB_BUCKET_NAME"] \
    --auth-domain local

echo "Couchbase initialization complete!"
