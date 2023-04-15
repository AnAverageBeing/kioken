#!/bin/bash

# Define output directory
OUTPUT_DIR="bin"

# Create output directory if it doesn't exist
if [ ! -d "$OUTPUT_DIR" ]; then
  mkdir "$OUTPUT_DIR"
fi

# Build kioken binary
go build -o "$OUTPUT_DIR/kioken" ./cmd/kioken

# Print success message
echo "Build successful! Binary can be found in $OUTPUT_DIR/kioken"
