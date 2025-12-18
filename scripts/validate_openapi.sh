#!/bin/bash

EXPECTED_DIR="test/integration/testdata/expected"
FILES=$(find "$EXPECTED_DIR" -type f \( -name "*_to_openapi_*" \) \( -name "*.json" -o -name "*.yaml" \))

echo "Validating OpenAPI files..."
echo "------------------------------------------------"

EXIT_CODE=0

for file in $FILES; do
    echo "Linting: $file"
    if ! npx --yes swagger-cli validate "$file"; then
        echo "FAILED: $file"
        EXIT_CODE=1
    else
        echo "PASSED: $file"
    fi
    echo "------------------------------------------------"
done

if [ $EXIT_CODE -eq 0 ]; then
    echo "All OpenAPI files passed validation!"
else
    echo "Some OpenAPI files failed validation."
fi

exit $EXIT_CODE
