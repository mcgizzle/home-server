#!/bin/bash

# List of directories with docker-compose.yml files
directories=(
  '/apps/dashboard'
)

for dir in "${directories[@]}"; do
  echo "Starting server in $dir"
  (cd "$dir" && docker-compose up -d)
done

echo "All projects started."
