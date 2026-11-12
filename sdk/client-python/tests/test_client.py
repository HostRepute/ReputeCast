from __future__ import annotations

import asyncio
import json

import httpx
import pytest

from reputecast import MAX_RETRY_DELAY, USER_AGENT, ReputeCast, ReputeCastError, RetryTransport, retry_delay, unwrap
from reputecast.api.audience import list_lists
from reputecast.api.team import list_members
from reputecast.api.transactional_messages import send_transactional_message
from reputecast.models import ItemsAudienceListSummary, TransactionalSend

BASE = "https://news.example.com"
LISTS = {"items": [{"id": "l1", "name": "Weekly Digest", "description": "", "doubleOptIn": False, "subscribed": 1, "pending": 0, "createdAt": "2026-01-01T00:00:00Z"}]}


class Server:
    """Answers with the given statuses in turn (the last one repeats) and records the requests."""

    def __init__(self, *answers: int | tuple[int, dict[str, str]], body: object = LISTS) -> None:
        self.answers = answers
        self.body = body
        self.requests: list[httpx.Request] = []

    def __call__(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        answer = self.answers[min(len(self.requests), len(self.answers)) - 1]
        status, headers = answer if isinstance(answer, tuple) else (answer, {})
        if status >= 400:
            return httpx.Response(status, headers=headers, json={"status": status, "title": "Busy", "detail": "Try again."})
        return httpx.Response(status, headers=headers, json=self.body)


def client(server: Server, delays: list[float], max_retries: int = 2) -> httpx.Client:
    transport = RetryTransport(max_retries, transport=httpx.MockTransport(server), sleep=delays.append)
    return httpx.Client(base_url=BASE, transport=transport)


def test_retries_a_get_with_backoff() -> None:
    server, delays = Server(503, 502, 200), []
    assert client(server, delays).get("/api/v1/x").status_code == 200
    assert len(server.requests) == 3
    assert 0.25 <= delays[0] <= 0.5
    assert 0.5 <= delays[1] <= 1.0


def test_honours_retry_after() -> None:
    assert retry_delay(0, "2") == 2
    assert retry_delay(0, "120") == MAX_RETRY_DELAY
    now = 1_762_855_200.0  # Tue, 11 Nov 2025 10:00:00 GMT
    assert retry_delay(0, "Tue, 11 Nov 2025 10:00:05 GMT", now=now) == 5
    assert retry_delay(0, "Tue, 11 Nov 2025 09:00:00 GMT", now=now) == 0
    assert retry_delay(0, "soon", rand=lambda: 0) == 0.25
    assert retry_delay(20, None, rand=lambda: 1) == MAX_RETRY_DELAY

    server, delays = Server((429, {"Retry-After": "3"}), 200), []
    assert client(server, delays).delete("/api/v1/x").status_code == 200
    assert delays == [3]


def test_retries_a_post_only_with_an_idempotency_key() -> None:
    server, delays = Server(503, 200), []
    assert client(server, delays).post("/api/v1/x", json={"a": 1}).status_code == 503
    assert len(server.requests) == 1

    server = Server(504, 200)
    res = client(server, delays).post("/api/v1/x", json={"a": 1}, headers={"Idempotency-Key": "k-1"})
    assert res.status_code == 200
    assert [r.content for r in server.requests] == [b'{"a":1}', b'{"a":1}']


def test_gives_up_and_never_retries_other_errors() -> None:
    server, delays = Server(429), []
    assert client(server, delays).get("/x").status_code == 429
    assert len(server.requests) == 3

    server = Server(500, 200)
    assert client(server, delays).get("/x").status_code == 500
    assert len(server.requests) == 1

    server = Server(503, 200)
    assert client(server, delays, max_retries=0).get("/x").status_code == 503
    assert len(server.requests) == 1


def test_never_resends_a_streamed_upload() -> None:
    server, delays = Server(503, 200), []
    res = client(server, delays).put("/x", content=iter([b"a,b\n", b"c,d\n"]))
    assert res.status_code == 503
    assert len(server.requests) == 1


def test_retries_asynchronously() -> None:
    server, delays = Server(503, 200), []

    async def sleep(seconds: float) -> None:
        delays.append(seconds)

    async def run() -> int:
        transport = RetryTransport(2, async_transport=httpx.MockTransport(server), async_sleep=sleep)
        async with httpx.AsyncClient(base_url=BASE, transport=transport) as c:
            return (await c.get("/x")).status_code

    assert asyncio.run(run()) == 200
    assert len(server.requests) == 2
    assert len(delays) == 1


def test_the_client_sends_the_key_and_user_agent() -> None:
    server = Server(200)
    rc = ReputeCast(api_key="rc_live_test", base_url=BASE + "/", transport=httpx.MockTransport(server))
    lists = unwrap(list_lists.sync_detailed(client=rc, workspace="acme-newsletter"))
    assert isinstance(lists, ItemsAudienceListSummary)
    assert lists.items[0].name == "Weekly Digest"
    sent = server.requests[0]
    assert sent.url == BASE + "/api/v1/workspaces/acme-newsletter/lists"
    assert sent.headers["Authorization"] == "Bearer rc_live_test"
    assert sent.headers["User-Agent"] == USER_AGENT
    assert USER_AGENT.startswith("ReputeCast-SDK-Python/")


def test_the_idempotency_key_lets_a_send_be_retried() -> None:
    server = Server(503, 202, body={"id": "0193", "status": "queued"})
    rc = ReputeCast("rc_live_test", BASE, transport=httpx.MockTransport(server))
    rc.get_httpx_client()._transport._sleep = lambda _: None  # type: ignore[attr-defined]
    msg = unwrap(
        send_transactional_message.sync_detailed(
            client=rc,
            workspace="acme-newsletter",
            idempotency_key="pwreset-u_42-t_7",
            body=TransactionalSend.from_dict({"template": "password-reset", "to": {"email": "sam.rivera@outlook.com"}}),
        )
    )
    assert msg.id == "0193"
    assert [r.headers["Idempotency-Key"] for r in server.requests] == ["pwreset-u_42-t_7"] * 2


def test_unwrap_raises_the_problem() -> None:
    problem = {
        "type": "about:blank",
        "status": 403,
        "title": "Forbidden",
        "detail": "You need team:view. Linh Tran can give it to you.",
        "code": "permission_denied",
        "permission": "team:view",
        "helpers": ["Linh Tran"],
        "errors": [{"location": "path.workspace", "message": "No."}],
        "meta": {"reference": "err_7KQ2M4XA"},
    }

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(403, headers={"Content-Type": "application/problem+json"}, content=json.dumps(problem).encode())

    rc = ReputeCast("rc_live_test", BASE, transport=httpx.MockTransport(handler))
    with pytest.raises(ReputeCastError) as err:
        unwrap(list_members.sync_detailed(client=rc, workspace="acme-newsletter"))
    e = err.value
    assert (e.status, e.code, e.permission, e.helpers) == (403, "permission_denied", "team:view", ["Linh Tran"])
    assert str(e) == e.detail == problem["detail"]
    assert e.title == "Forbidden"
    assert e.errors == problem["errors"]
    assert e.meta == {"reference": "err_7KQ2M4XA"}


def test_unwrap_explains_an_answer_that_isnt_a_problem() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(502, text="Bad gateway from the proxy")

    rc = ReputeCast("rc_live_test", BASE, max_retries=0, transport=httpx.MockTransport(handler))
    with pytest.raises(ReputeCastError) as err:
        unwrap(list_lists.sync_detailed(client=rc, workspace="acme-newsletter"))
    assert (err.value.status, err.value.code, err.value.detail, err.value.title) == (502, "", "Bad gateway from the proxy", "Bad Gateway")


def test_refuses_options_that_cant_work() -> None:
    with pytest.raises(ValueError):
        ReputeCast("", BASE)
    with pytest.raises(ValueError):
        ReputeCast("k", "")
    with pytest.raises(ValueError):
        ReputeCast("k", BASE, max_retries=-1)
