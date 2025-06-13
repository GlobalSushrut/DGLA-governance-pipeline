#!/bin/bash
# DGLA IP Protection Pre-Receive Hook
# This hook runs on the server side before accepting any push

# Get the repository name
REPO_NAME=$(basename "$(pwd)")

# Set timestamp
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Get user information
USER_EMAIL=$(git config user.email)
USER_NAME=$(git config user.name)
IP_ADDRESS=$(echo $SSH_CONNECTION | awk '{print $1}')

# Prepare log entry
LOG_ENTRY="{\"timestamp\":\"$TIMESTAMP\",\"repo\":\"$REPO_NAME\",\"user\":\"$USER_NAME\",\"email\":\"$USER_EMAIL\",\"ip\":\"$IP_ADDRESS\",\"action\":\"clone\"}"

# Log the clone event
echo "$LOG_ENTRY" >> /path/to/security/clone_logs.json

# Read stdin (ref updates)
while read oldrev newrev refname; do
  # Skip if this is a new branch or tag
  if [ "$oldrev" = "0000000000000000000000000000000000000000" ]; then
    continue
  fi

  # Check for license file removal/modification
  git diff --name-only "$oldrev" "$newrev" | grep -q "LICENSE.json"
  if [ $? -eq 0 ]; then
    echo "ERROR: Modifying the LICENSE.json file is not permitted."
    echo "Your changes include modifications to protected IP licensing information."
    echo "Please restore the original LICENSE.json file to proceed."
    exit 1
  fi

  # Check for IP watermark removal
  git diff --name-only "$oldrev" "$newrev" | grep -q "DGLA-IP-WATERMARK"
  if [ $? -eq 0 ]; then
    echo "WARNING: Detected potential tampering with IP watermarks."
    echo "These watermarks are essential for tracking and protecting intellectual property."
    echo "This activity has been logged and may be reviewed by the security team."
  fi
done

# Insert a unique watermark into the cloned repository
WATERMARK=$(echo "$USER_NAME:$USER_EMAIL:$TIMESTAMP" | sha256sum | cut -d' ' -f1)
echo "# DGLA-IP-WATERMARK: $WATERMARK" >> .git/DGLA-CLONE-SIGNATURE

# Call webhook to record clone event
if command -v curl &> /dev/null; then
  curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"user\":\"$USER_NAME\",\"email\":\"$USER_EMAIL\",\"timestamp\":\"$TIMESTAMP\",\"watermark\":\"$WATERMARK\",\"ip\":\"$IP_ADDRESS\"}" \
    https://api.dgla-secure.com/webhooks/clone-tracking || true
fi

exit 0
