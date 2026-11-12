"""The client factory and its transport: retries, and problems for error pages."""

from __future__ import annotations

import asyncio
import email.utils
import json
import random
import time
from collections.abc import Awaitable, Callable
from datetime import timezone
from http import HTTPStatus

import httpx

from ._version import __version__
from .client import AuthenticatedClient

USER_AGENT = f"ReputeCast-SDK-Python/{__version__}"
"""The User-Agent of every request: Settings › Integrations names the caller by it."""

RETRY_STATUSES = frozenset({429, 502, 503, 504})
"""The answers a retry may fix: rate limited, or the server briefly unavailable."""

MAX_RETRY_DELAY = 30.0
"""The longest wait between two attempts, in seconds, whatever Retry-After asks."""

_BASE_DELAY = 0.5
_IDEMPOTENT_METHODS = frozenset({"GET", "HEAD", "PUT", "DELETE"})


def is_retryable(request: httpx.Request) -> bool:
    """Whether a request may be sent again.

    Its method is idempotent, or it is a POST with an Idempotency-Key (the
    server answers a repeat with the first result instead of acting twice),
    and its body is in memory (a streamed upload can't be sent twice).
    """
    if request.method not in _IDEMPOTENT_METHODS and not (request.method == "POST" and "Idempotency-Key" in request.headers):
        return False
    return isinstance(request.stream, httpx.ByteStream)


def retry_delay(attempt: int, retry_after: str | None, now: float | None = None, rand: Callable[[], float] = random.random) -> float:
    """How long to wait, in seconds, before retry number ``attempt`` (from 0).

    The answer's Retry-After (seconds or an HTTP date) when it has one, else
    exponential backoff from 0.5 s with jitter; never more than 30 s.
    """
    if retry_after:
        value = retry_after.strip()
        seconds: float | None = None
        if value.isascii() and value.isdigit():
            seconds = float(value)
        else:
            try:
                when = email.utils.parsedate_to_datetime(value)
            except (TypeError, ValueError):
                when = None
            if when is not None:
                if when.tzinfo is None:
                    when = when.replace(tzinfo=timezone.utc)
                seconds = max(0.0, when.timestamp() - (time.time() if now is None else now))
        if seconds is not None:
            return min(seconds, MAX_RETRY_DELAY)
    ceiling = min(_BASE_DELAY * 2**attempt, MAX_RETRY_DELAY)
    return ceiling / 2 + rand() * ceiling / 2


class RetryTransport(httpx.BaseTransport, httpx.AsyncBaseTransport):
    """Sends a request again after a 429, 502, 503 or 504, up to ``max_retries`` times.

    An error answer that isn't JSON (a proxy's page) reaches the client as a
    problem whose detail is its text, so the generated parsers and unwrap
    read it as any other. It serves both httpx.Client and httpx.AsyncClient,
    over ``transport`` and ``async_transport`` (httpx's own by default, made
    when first used).
    """

    def __init__(
        self,
        max_retries: int = 2,
        *,
        transport: httpx.BaseTransport | None = None,
        async_transport: httpx.AsyncBaseTransport | None = None,
        sleep: Callable[[float], None] = time.sleep,
        async_sleep: Callable[[float], Awaitable[None]] = asyncio.sleep,
    ) -> None:
        if max_retries < 0:
            raise ValueError("max_retries is a whole number, 0 or more")
        self.max_retries = max_retries
        self._transport = transport
        self._async_transport = async_transport
        self._sleep = sleep
        self._async_sleep = async_sleep

    def handle_request(self, request: httpx.Request) -> httpx.Response:
        if self._transport is None:
            self._transport = httpx.HTTPTransport()
        # Decided before the first attempt: sending reads a streamed body.
        retries = self.max_retries if is_retryable(request) else 0
        attempt = 0
        while True:
            response = self._transport.handle_request(request)
            if attempt >= retries or response.status_code not in RETRY_STATUSES:
                break
            delay = retry_delay(attempt, response.headers.get("Retry-After"))
            response.read()
            response.close()
            self._sleep(delay)
            attempt += 1
        if not _is_error_page(response):
            return response
        response.read()
        response.close()
        return _as_problem(response)

    async def handle_async_request(self, request: httpx.Request) -> httpx.Response:
        if self._async_transport is None:
            self._async_transport = httpx.AsyncHTTPTransport()
        retries = self.max_retries if is_retryable(request) else 0
        attempt = 0
        while True:
            response = await self._async_transport.handle_async_request(request)
            if attempt >= retries or response.status_code not in RETRY_STATUSES:
                break
            delay = retry_delay(attempt, response.headers.get("Retry-After"))
            await response.aread()
            await response.aclose()
            await self._async_sleep(delay)
            attempt += 1
        if not _is_error_page(response):
            return response
        await response.aread()
        await response.aclose()
        return _as_problem(response)

    def close(self) -> None:
        if self._transport is not None:
            self._transport.close()

    async def aclose(self) -> None:
        if self._async_transport is not None:
            await self._async_transport.aclose()


def _is_error_page(response: httpx.Response) -> bool:
    """An error answer the generated parsers can't read: not JSON (a proxy's page, an empty body)."""
    if response.status_code < 400:
        return False
    media = response.headers.get("Content-Type", "").split(";", 1)[0].strip().lower()
    return not (media == "application/json" or media.endswith("+json"))


def _as_problem(response: httpx.Response) -> httpx.Response:
    """The problem an error page stands for: its status, and its text as the detail."""
    try:
        title = HTTPStatus(response.status_code).phrase
    except ValueError:
        title = ""
    problem = {"type": "about:blank", "status": response.status_code, "title": title}
    text = response.text.strip()[:500]
    if text:
        problem["detail"] = text
    headers = [(k, v) for k, v in response.headers.multi_items() if k.lower() not in ("content-type", "content-length", "content-encoding")]
    headers.append(("Content-Type", "application/problem+json"))
    return httpx.Response(response.status_code, headers=headers, content=json.dumps(problem).encode(), extensions=response.extensions)



def ReputeCast(  # noqa: N802 - reads as the client it makes
    api_key: str,
    base_url: str,
    timeout: float = 30,
    max_retries: int = 2,
    *,
    transport: httpx.BaseTransport | None = None,
    async_transport: httpx.AsyncBaseTransport | None = None,
) -> AuthenticatedClient:
    """Returns a client for a ReputeCast instance, for every function in reputecast.api.

    ``api_key`` is an API key (rc_live_…) or an MCP agent token (rc_agent_…);
    ``base_url`` the instance's URL, e.g. https://news.example.com.
    ``timeout`` is in seconds. GET, HEAD, PUT and DELETE requests, and POSTs
    with an Idempotency-Key, are sent again up to ``max_retries`` times after
    a 429, 502, 503 or 504, waiting as Retry-After asks (at most 30 s) or
    with exponential backoff. ``transport`` and ``async_transport`` replace
    httpx's own under the retries (a proxy, custom TLS, tests).

        rc = ReputeCast(api_key=os.environ["REPUTECAST_KEY"], base_url="https://news.example.com")
        lists = unwrap(list_lists.sync_detailed(client=rc, workspace="acme-newsletter"))
    """
    if not api_key:
        raise ValueError("api_key is empty: create a key in Settings › Keys & MCP")
    if not base_url:
        raise ValueError("base_url is the instance's URL, e.g. https://news.example.com")
    return AuthenticatedClient(
        base_url=base_url.rstrip("/"),
        token=api_key,
        timeout=httpx.Timeout(timeout),
        headers={"User-Agent": USER_AGENT},
        httpx_args={"transport": RetryTransport(max_retries, transport=transport, async_transport=async_transport)},
    )
