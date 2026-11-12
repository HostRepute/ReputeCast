from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_batch_schedule_input import TransactionalBatchScheduleInput
from ...models.transactional_batch_schedule_result import TransactionalBatchScheduleResult
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: TransactionalBatchScheduleInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/transactional/batches/cancel".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalBatchScheduleResult:
    if response.status_code == 200:
        response_200 = TransactionalBatchScheduleResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalBatchScheduleResult]:
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
    body: TransactionalBatchScheduleInput,

) -> Response[Problem | TransactionalBatchScheduleResult]:
    """ Cancel every message of a batch that still waits for its send time

     The batch is the call's Idempotency-Key. Messages already on their way, sent or cancelled are
    counted as skipped. Each message is audited. Review 2026-11 TX-7.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (TransactionalBatchScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalBatchScheduleResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalBatchScheduleInput,

) -> Problem | TransactionalBatchScheduleResult | None:
    """ Cancel every message of a batch that still waits for its send time

     The batch is the call's Idempotency-Key. Messages already on their way, sent or cancelled are
    counted as skipped. Each message is audited. Review 2026-11 TX-7.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (TransactionalBatchScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalBatchScheduleResult
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalBatchScheduleInput,

) -> Response[Problem | TransactionalBatchScheduleResult]:
    """ Cancel every message of a batch that still waits for its send time

     The batch is the call's Idempotency-Key. Messages already on their way, sent or cancelled are
    counted as skipped. Each message is audited. Review 2026-11 TX-7.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (TransactionalBatchScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalBatchScheduleResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalBatchScheduleInput,

) -> Problem | TransactionalBatchScheduleResult | None:
    """ Cancel every message of a batch that still waits for its send time

     The batch is the call's Idempotency-Key. Messages already on their way, sent or cancelled are
    counted as skipped. Each message is audited. Review 2026-11 TX-7.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (TransactionalBatchScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalBatchScheduleResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
