from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_send import TransactionalSend
from ...models.transactional_send_result import TransactionalSendResult
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: TransactionalSend,
    idempotency_key: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(idempotency_key, Unset):
        headers["Idempotency-Key"] = idempotency_key



    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/transactional/messages".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalSendResult:
    if response.status_code == 202:
        response_202 = TransactionalSendResult.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalSendResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalSend,
    idempotency_key: str | Unset = UNSET,

) -> Response[Problem | TransactionalSendResult]:
    """ Send one message with a template's live version

     Answers 202 with the message ID and status queued (scheduled with send_at). Errors are RFC 9457
    problems whose detail says what to fix: template_not_found, template_not_published,
    missing_variables (listed in missing), invalid_recipient, recipient_suppressed (with the reason and
    date), sender_not_verified, attachments_too_large (naming the route's provider limit), rate_limited
    (with Retry-After), no_healthy_route. Send an Idempotency-Key header: a retry with the same key
    within 24 hours gets the first answer and sends nothing. A test-mode key delivers only to the
    workspace's members and its allowed test addresses.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry with the same key within 24 hours never sends twice
        body (TransactionalSend):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalSendResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,
idempotency_key=idempotency_key,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalSend,
    idempotency_key: str | Unset = UNSET,

) -> Problem | TransactionalSendResult | None:
    """ Send one message with a template's live version

     Answers 202 with the message ID and status queued (scheduled with send_at). Errors are RFC 9457
    problems whose detail says what to fix: template_not_found, template_not_published,
    missing_variables (listed in missing), invalid_recipient, recipient_suppressed (with the reason and
    date), sender_not_verified, attachments_too_large (naming the route's provider limit), rate_limited
    (with Retry-After), no_healthy_route. Send an Idempotency-Key header: a retry with the same key
    within 24 hours gets the first answer and sends nothing. A test-mode key delivers only to the
    workspace's members and its allowed test addresses.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry with the same key within 24 hours never sends twice
        body (TransactionalSend):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalSendResult
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,
idempotency_key=idempotency_key,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalSend,
    idempotency_key: str | Unset = UNSET,

) -> Response[Problem | TransactionalSendResult]:
    """ Send one message with a template's live version

     Answers 202 with the message ID and status queued (scheduled with send_at). Errors are RFC 9457
    problems whose detail says what to fix: template_not_found, template_not_published,
    missing_variables (listed in missing), invalid_recipient, recipient_suppressed (with the reason and
    date), sender_not_verified, attachments_too_large (naming the route's provider limit), rate_limited
    (with Retry-After), no_healthy_route. Send an Idempotency-Key header: a retry with the same key
    within 24 hours gets the first answer and sends nothing. A test-mode key delivers only to the
    workspace's members and its allowed test addresses.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry with the same key within 24 hours never sends twice
        body (TransactionalSend):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalSendResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,
idempotency_key=idempotency_key,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalSend,
    idempotency_key: str | Unset = UNSET,

) -> Problem | TransactionalSendResult | None:
    """ Send one message with a template's live version

     Answers 202 with the message ID and status queued (scheduled with send_at). Errors are RFC 9457
    problems whose detail says what to fix: template_not_found, template_not_published,
    missing_variables (listed in missing), invalid_recipient, recipient_suppressed (with the reason and
    date), sender_not_verified, attachments_too_large (naming the route's provider limit), rate_limited
    (with Retry-After), no_healthy_route. Send an Idempotency-Key header: a retry with the same key
    within 24 hours gets the first answer and sends nothing. A test-mode key delivers only to the
    workspace's members and its allowed test addresses.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry with the same key within 24 hours never sends twice
        body (TransactionalSend):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalSendResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,
idempotency_key=idempotency_key,

    )).parsed
