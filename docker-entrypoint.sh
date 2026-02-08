#!/bin/sh
# Development entrypoint

echo "Starting product-service with Air..."
cd /app/omnipos-product-service
exec air -c .air.toml
