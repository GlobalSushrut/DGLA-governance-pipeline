#!/bin/bash
# Update the namespace in all infrastructure YAML files

OLD_NAMESPACE="dgla"
NEW_NAMESPACE="rogers-5g-production"

echo "Updating namespace from $OLD_NAMESPACE to $NEW_NAMESPACE in all YAML files..."

# Find all YAML files in the infrastructure directory
YAML_FILES=$(find /home/umesh/Documents/DGLA_progects/data-governance-pipeline/infrastructure -type f -name "*.yaml")

# Update namespace in each file
for file in $YAML_FILES; do
  echo "Processing file: $file"
  # Using sed to replace namespace values
  sed -i "s/namespace: $OLD_NAMESPACE/namespace: $NEW_NAMESPACE/g" "$file"
  sed -i "s/name: $OLD_NAMESPACE/name: $NEW_NAMESPACE/g" "$file"
done

echo "Updated namespace in all YAML files."

# Also update the use-cases YAML files for Rogers 5G
echo "Updating Rogers 5G specific YAML files..."
ROGERS_YAML_FILES=$(find /home/umesh/Documents/DGLA_progects/data-governance-pipeline/use-cases/rogers-5g -type f -name "*.yaml")

for file in $ROGERS_YAML_FILES; do
  echo "Processing file: $file"
  sed -i "s/namespace: $OLD_NAMESPACE/namespace: $NEW_NAMESPACE/g" "$file"
done

echo "Namespace update completed!"
