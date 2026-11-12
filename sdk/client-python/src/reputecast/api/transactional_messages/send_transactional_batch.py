from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_batch import TransactionalBatch
from ...models.transactional_batch_result import TransactionalBatchResult
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: TransactionalBatch,
    idempotency_key: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(idempotency_key, Unset):
        headers["Idempotency-Key"] = idempotency_key



    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/transactional/messages/batch".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalBatchResult:
    if response.status_code == 200:
        response_200 = TransactionalBatchResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalBatchResult]:
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
    body: TransactionalBatch,
    idempotency_key: str | Unset = UNSET,

) -> Response[Problem | TransactionalBatchResult]:
    """ Send up to 100 messages, each with its own recipient and data

     Each message gets its own answer in results, in the order sent: its ID and status, or the problem
    that stopped it (the same codes as one send).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry of the same batch within 24 hours never sends twice
        body (TransactionalBatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalBatchResult]
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
    body: TransactionalBatch,
    idempotency_key: str | Unset = UNSET,

) -> Problem | TransactionalBatchResult | None:
    """ Send up to 100 messages, each with its own recipient and data

     Each message gets its own answer in results, in the order sent: its ID and status, or the problem
    that stopped it (the same codes as one send).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry of the same batch within 24 hours never sends twice
        body (TransactionalBatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalBatchResult
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
    body: TransactionalBatch,
    idempotency_key: str | Unset = UNSET,

) -> Response[Problem | TransactionalBatchResult]:
    """ Send up to 100 messages, each with its own recipient and data

     Each message gets its own answer in results, in the order sent: its ID and status, or the problem
    that stopped it (the same codes as one send).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry of the same batch within 24 hours never sends twice
        body (TransactionalBatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalBatchResult]
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
    body: TransactionalBatch,
    idempotency_key: str | Unset = UNSET,

) -> Problem | TransactionalBatchResult | None:
    """ Send up to 100 messages, each with its own recipient and data

     Each message gets its own answer in results, in the order sent: its ID and status, or the problem
    that stopped it (the same codes as one send).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): A retry of the same batch within 24 hours never sends twice
        body (TransactionalBatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalBatchResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,
idempotency_key=idempotency_key,

    )).parsed
