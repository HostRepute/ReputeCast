"""Verify the webhooks ReputeCast delivers to your endpoints.

Each delivery is a JSON POST signed with the endpoint's secret (whsec_…):
X-ReputeCast-Signature holds the hex HMAC-SHA256 of "<timestamp>.<raw body>"
(two comma-separated signatures for 24 hours after the secret is rotated)
and X-ReputeCast-Timestamp the Unix time in seconds.

    from reputecast import webhooks

    webhooks.verify(os.environ["REPUTECAST_WEBHOOK_SECRET"], request.get_data(), request.headers)
"""

from __future__ import annotations

import hashlib
import hmac
import time
from collections.abc import Iterable, Mapping
from datetime import datetime
from typing import Any, Literal

SIGNATURE_HEADER = "X-ReputeCast-Signature"
TIMESTAMP_HEADER = "X-ReputeCast-Timestamp"
DEFAULT_TOLERANCE_SECONDS = 300

Reason = Literal["missing_header", "timestamp", "signature"]


class WebhookVerificationError(Exception):
    """A delivery that isn't from ReputeCast, was changed on the way or is too old: answer it with 400 or 401."""

    def __init__(self, reason: Reason, message: str) -> None:
        self.reason: Reason = reason
        super().__init__(message)


def verify(
    secrets: str | Iterable[str],
    body: bytes | bytearray | memoryview | str,
    headers: Mapping[str, Any],
    tolerance_seconds: float = DEFAULT_TOLERANCE_SECONDS,
    now: float | datetime | None = None,
) -> None:
    """Checks that a delivery was signed by ReputeCast with one of ``secrets``.

    ``body`` is the raw request body, exactly as received (not parsed and
    serialised again); ``headers`` the request's headers, looked up without
    regard to case. The timestamp must be within ``tolerance_seconds`` of
    ``now`` (the current time by default). Signatures are compared in
    constant time. Raises WebhookVerificationError when the delivery fails.
    """
    timestamp = _header(headers, TIMESTAMP_HEADER)
    signature = _header(headers, SIGNATURE_HEADER)
    if not timestamp or not signature:
        missing = SIGNATURE_HEADER if timestamp else TIMESTAMP_HEADER
        raise WebhookVerificationError("missing_header", f"The delivery has no {missing} header.")
    timestamp = timestamp.strip()
    if not (timestamp.isascii() and timestamp.isdigit()):
        raise WebhookVerificationError("timestamp", f"{TIMESTAMP_HEADER} isn't a Unix time in seconds.")
    current = time.time() if now is None else now.timestamp() if isinstance(now, datetime) else float(now)
    if abs(current - int(timestamp)) > tolerance_seconds:
        raise WebhookVerificationError("timestamp", f"The delivery's timestamp is more than {tolerance_seconds:g} seconds from now.")

    raw = body.encode() if isinstance(body, str) else bytes(body)
    signed = timestamp.encode() + b"." + raw
    candidates = [s.strip().encode() for s in signature.split(",")]
    for secret in [secrets] if isinstance(secrets, str) else secrets:
        if not secret:
            continue
        want = hmac.new(secret.encode(), signed, hashlib.sha256).hexdigest().encode()
        for candidate in candidates:
            if hmac.compare_digest(want, candidate):
                return
    raise WebhookVerificationError("signature", "No signature of the delivery matches the signing secret.")


def _header(headers: Mapping[str, Any], name: str) -> str | None:
    value = headers.get(name)
    if value is None:
        lower = name.lower()
        for key, v in headers.items():
            if key.lower() == lower:
                value = v
                break
    if value is None:
        return None
    if isinstance(value, (list, tuple)):
        return ",".join(str(v) for v in value)
    return str(value)
