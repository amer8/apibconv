#!/bin/bash

FILES=$(find test/integration/testdata/expected -name "*to_asyncapi_*")

for f in $FILES; do
  echo "Validating $f..."
  if npx --yes @asyncapi/cli validate "$f" > /dev/null 2>&1; then
    echo "  PASS"
  else
    echo "  FAIL"
    npx --yes @asyncapi/cli validate "$f"
  fi
done
