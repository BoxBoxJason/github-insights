#!/bin/sh
# Bundle ../results/*.json into data.js so the static viewer can load it
# without a server (works over file://). JS array literals tolerate the
# trailing comma, so no comma-joining logic is needed.
set -e

cd "$(dirname "$0")"
out=data.js

printf 'window.REPORTS = [\n' > "$out"
count=0
for f in ../results/*.json; do
  [ -e "$f" ] || continue
  cat "$f" >> "$out"
  printf ',\n' >> "$out"
  count=$((count + 1))
done
printf '];\n' >> "$out"

echo "Wrote $out from $count report(s) in results/*.json"
