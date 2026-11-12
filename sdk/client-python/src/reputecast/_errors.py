"""ReputeCastError and unwrap: the API's RFC 9457 problems as exceptions."""

from __future__ import annotations

import json
from collections.abc import Mapping, MutableMapping
from http import HTTPStatus
from typing import Any, TypeVar, cast

from .models.problem import Problem
from .types import Response

T = TypeVar("T")


class ReputeCastError(Exception):
    """An error answer of the API (an RFC 9457 problem).

    ``status`` is the HTTP status; ``code`` the stable code
    (permission_denied, validation_failed, not_found…); ``detail`` the
    sentence that explains it; ``errors`` the fields at fault; and, for a
    missing permission, ``permission`` and the ``helpers`` who hold it.
    """

    def __init__(self, status: int, problem: Mapping[str, Any] | None = None, headers: MutableMapping[str, str] | None = None) -> None:
        p = problem or {}
        self.status = status
        self.code: str = str(p.get("code") or "")
        self.title: str = str(p.get("title") or "")
        self.detail: str = str(p.get("detail") or "")
        self.type: str = str(p.get("type") or "")
        self.errors: list[dict[str, Any]] = [e for e in p.get("errors") or [] if isinstance(e, dict)]
        self.permission: str = str(p.get("permission") or "")
        self.helpers: list[str] = [str(h) for h in p.get("helpers") or []]
        self.meta: dict[str, Any] = dict(p.get("meta") or {})
        self.headers = headers
        super().__init__(self.detail or self.title or f"The API answered HTTP {status}.")

    @classmethod
    def from_response(cls, status: int, content: bytes, headers: MutableMapping[str, str] | None = None) -> ReputeCastError:
        """The error for an answer whose body is a problem, other JSON or text."""
        try:
            body = json.loads(content) if content else None
        except ValueError:
            body = None
        if isinstance(body, dict):
            return cls(status, body, headers)
        try:
            title = HTTPStatus(status).phrase
        except ValueError:
            title = ""
        text = content.decode("utf-8", "replace").strip()[:500]
        return cls(status, {"title": title, "detail": text}, headers)

    def __repr__(self) -> str:
        return f"ReputeCastError(status={self.status}, code={self.code!r}, detail={self.detail!r})"


def unwrap(response: Response[T | Problem] | Response[T]) -> T:
    """Returns the parsed body of a ``sync_detailed``/``asyncio_detailed`` call, or raises ReputeCastError.

    An answer without a body (204) returns None.
    """
    status = int(response.status_code)
    if 200 <= status < 300:
        return cast(T, response.parsed)
    raise ReputeCastError.from_response(status, response.content, response.headers)
