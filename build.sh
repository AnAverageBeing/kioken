#!/bin/bash

# Define output directory
OUTPUT_DIR="."

# Create output directory if it doesn't exist
if [ ! -d "$OUTPUT_DIR" ]; then
  mkdir "$OUTPUT_DIR"
fi

# Build kioken binary and capture output and error streams
BUILD_OUTPUT=$(go build -ldflags="-s -w" -tags netgo -ldflags="-extldflags=-static" -o "$OUTPUT_DIR/kioken" ./cmd/kioken 2>&1)

# Check if there were any errors during build
if [ $? -ne 0 ]; then
  # Print error message in red color and exit script
  echo -e "\033[31mBuild failed with following error:\033[0m"
  echo -e "$BUILD_OUTPUT"
  exit 1
else
  # Print success message in green color
  echo -e "\033[32mBuild successful! Binary can be found in $OUTPUT_DIR/kioken\033[0m"
fi
