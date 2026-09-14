#!/usr/bin/env bash

set -euo pipefail

# 1. Validate input parameter
if [ -z "${1:-}" ]; then
  echo "Error: Missing image name."
  echo "Usage: $0 <image-name> [input-file] [output-file]"
  exit 1
fi

IMAGE_NAME="$1"
INPUT_FILE="${2:-docker-compose.yml}"
OUTPUT_FILE="${3:-docker-compose.override.yml}"

# 2. Check if input docker-compose file exists
if [ ! -f "$INPUT_FILE" ]; then
  echo "Error: Input file '$INPUT_FILE' not found."
  exit 1
fi

# 3. Generate override file using python yq (passing parameter safely via --arg)
yq -y --arg img "$IMAGE_NAME" \
  '.services |= with_entries(.value = {"image": $img})' \
  "$INPUT_FILE" > "$OUTPUT_FILE"

# 4. Verification step
if [ -s "$OUTPUT_FILE" ]; then
  echo "Successfully generated '$OUTPUT_FILE' with image: $IMAGE_NAME"
else
  echo "Error: Failed to create '$OUTPUT_FILE' or output file is empty."
  exit 1
fi
