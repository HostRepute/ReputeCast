from __future__ import annotations

import json
from pathlib import Path

import pytest

from reputecast import webhooks

# Signed by the server's webhooks.Sign (internal/webhooks/vectors_test.go).
VECTORS = json.loads((Path(__file__).resolve().parents[2] / "testdata" / "webhook-vectors.json").read_text(encoding="utf-8"))


def test_the_vectors_are_there() -> None:
    assert len(VECTORS["vectors"]) > 10
    assert VECTORS["toleranceSeconds"] == webhooks.DEFAULT_TOLERANCE_SECONDS


@pytest.mark.parametrize("vector", VECTORS["vectors"], ids=[v["name"] for v in VECTORS["vectors"]])
def test_vector(vector: dict) -> None:
    def check() -> None:
        webhooks.verify(vector["secrets"], vector["body"], vector["headers"], now=vector["now"])

    if vector["valid"]:
        check()
    else:
        with pytest.raises(webhooks.WebhookVerificationError) as err:
            check()
        assert err.value.reason == vector["error"]


def test_bytes_one_secret_and_a_datetime() -> None:
    from datetime import datetime, timezone

    v = next(x for x in VECTORS["vectors"] if x["valid"] and len(x["secrets"]) == 1)
    webhooks.verify(v["secrets"][0], v["body"].encode(), v["headers"], now=datetime.fromtimestamp(v["now"], tz=timezone.utc))
    webhooks.verify(iter(v["secrets"]), bytearray(v["body"].encode()), {k.upper(): val for k, val in v["headers"].items()}, now=v["now"])


def test_the_tolerance_given() -> None:
    v = next(x for x in VECTORS["vectors"] if x["name"] == "expired timestamp")
    webhooks.verify(v["secrets"], v["body"], v["headers"], tolerance_seconds=600, now=v["now"])
