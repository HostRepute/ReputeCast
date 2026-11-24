#!/usr/bin/env bash
# Checks the licensing of the repository (LICENSING.md): every file has a
# license (reuse lint, REUSE.toml), and every published package carries its
# own LICENSE with the license REUSE.toml gives it, in its metadata and in
# what the registries receive. Used by `make licenses` and CI.
set -euo pipefail
cd "$(dirname "$0")/.."

command -v reuse >/dev/null || { echo "install reuse 6: pipx install reuse (https://reuse.software)"; exit 1; }
python3 -c 'import build' 2>/dev/null || { echo "install build: python3 -m pip install build"; exit 1; }

reuse lint

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
reuse spdx > "$tmp/reuse.spdx"
failed=0
fail() { echo "✗ $*" >&2; failed=1; }

# The license REUSE.toml gives a file.
license_of() {
  awk -v f="FileName: ./$1" '$0 == f { found = 1; next } found && /^FileName: / { exit } found && /^LicenseInfoInFile: / { print $2; exit }' "$tmp/reuse.spdx"
}

# A package's LICENSE must be the text in LICENSES/, with HostRepute's
# copyright: npm packs it, PyPI wheels and sdists carry it (PEP 639 can't
# reach a parent directory), and the go command copies the repository's
# root LICENSE (AGPL) into a module that has none.
check_package() { # dir, metadata file, the license the metadata declares
  local dir=$1 meta=$2 declared=$3 want file=$1/LICENSE
  [ -f "$file" ] || [ ! -f "$dir/LICENSE.md" ] || file=$dir/LICENSE.md
  want=$(license_of "$dir/$meta")
  [ -n "$want" ] || { fail "$dir/$meta: no license in REUSE.toml"; return; }
  [ "$declared" = "$want" ] || fail "$dir/$meta declares $declared; REUSE.toml says $want"
  if [ ! -f "$file" ]; then
    fail "$file is missing"
  elif ! diff -q <(sed -E 's/^Copyright (\(c\) )?(2026 HostRepute)$/Copyright \1<HOLDER>/' "$file") \
    <(sed -E 's/^Copyright (\(c\) )?(<year> <copyright holders>|<YEAR> <COPYRIGHT HOLDER>)$/Copyright \1<HOLDER>/' "LICENSES/$want.txt") >/dev/null; then
    fail "$file isn't LICENSES/$want.txt with HostRepute's copyright"
  fi
}

npm_license() { node -p "require('./$1/package.json').license"; }
for dir in sdk/client-ts sdk/adapter-sdk sdk/adapter-template integrations/zapier integrations/make integrations/n8n; do
  check_package "$dir" package.json "$(npm_license "$dir")"
done
check_package sdk/client-python pyproject.toml "$(python3 -c 'import tomllib; print(tomllib.load(open("sdk/client-python/pyproject.toml", "rb"))["project"]["license"])')"
# A Go module has no license field: its LICENSE is all there is.
check_package sdk/client-go go.mod "$(license_of sdk/client-go/go.mod)"

# What the registries receive: npm publishes @reputecast/client and the n8n
# nodes.
for dir in sdk/client-ts integrations/n8n; do
  (cd "$dir" && npm pack --dry-run --json 2>/dev/null) |
    node -e 'const [p] = JSON.parse(require("fs").readFileSync(0, "utf8")); process.exit(p.files.some((f) => /^LICENSE(\.md)?$/.test(f.path)) ? 0 : 1)' ||
    fail "$dir: npm pack leaves out its license"
done
python3 -m build --outdir "$tmp/dist" sdk/client-python >"$tmp/build.log" 2>&1 || { cat "$tmp/build.log"; exit 1; }
for f in "$tmp"/dist/*.whl "$tmp"/dist/*.tar.gz; do
  python3 - "$f" <<'PY' || fail "$(basename "$f"): no MIT License-Expression or LICENSE"
import sys, tarfile, zipfile
f = sys.argv[1]
if f.endswith(".whl"):
    z = zipfile.ZipFile(f)
    names = z.namelist()
    meta = z.read(next(n for n in names if n.endswith(".dist-info/METADATA"))).decode()
else:
    t = tarfile.open(f)
    names = t.getnames()
    meta = t.extractfile(next(n for n in names if n.count("/") == 1 and n.endswith("/PKG-INFO"))).read().decode()
ok = "License-Expression: MIT\n" in meta and "License-File: LICENSE\n" in meta
ok = ok and any(n.endswith("/LICENSE") for n in names)
sys.exit(0 if ok else 1)
PY
done

[ "$failed" = 0 ] && echo "✓ every published package carries its license"
exit "$failed"
