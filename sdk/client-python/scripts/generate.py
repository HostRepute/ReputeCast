"""Regenerates the API modules of the reputecast package.

openapi-python-client (pinned in pyproject.toml's dev group) generates a
whole package from sdk/openapi/public-3.0.json into a temporary folder;
this replaces the package's generated parts (api/, models/, client.py,
errors.py and types.py) with them and leaves the hand-written modules
(__init__.py, _client.py, _errors.py, _version.py, webhooks.py, py.typed)
as they are, so the public import paths are reputecast.api.<tag>.<operation>
and reputecast.models.

Run it with the virtual environment's Python: `make sdks` does.
"""

from __future__ import annotations

import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SPEC = ROOT.parent / "openapi" / "public-3.0.json"
CONFIG = ROOT / "openapi-python-client.yml"
PACKAGE = ROOT / "src" / "reputecast"
GENERATED = ("api", "models", "client.py", "errors.py", "types.py")


def main() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        out = Path(tmp) / "reputecast"
        subprocess.run(
            [
                sys.executable,
                "-m",
                "openapi_python_client",
                "generate",
                "--path",
                str(SPEC),
                "--config",
                str(CONFIG),
                "--meta",
                "none",
                "--output-path",
                str(out),
                "--fail-on-warning",
            ],
            check=True,
        )
        for name in GENERATED:
            src, dst = out / name, PACKAGE / name
            if not src.exists():
                sys.exit(f"openapi-python-client wrote no {name}")
            if dst.is_dir():
                shutil.rmtree(dst)
            elif dst.exists():
                dst.unlink()
            if src.is_dir():
                shutil.copytree(src, dst)
            else:
                shutil.copy2(src, dst)


if __name__ == "__main__":
    main()
