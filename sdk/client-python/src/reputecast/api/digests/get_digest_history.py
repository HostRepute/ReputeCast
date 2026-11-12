from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.digest_history import DigestHistory
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    digest_id: UUID,
    *,
    limit: int | Unset = 5,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/digests/{digest_id}/history".format(workspace=quote(str(workspace), safe=""),digest_id=quote(str(digest_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DigestHistory | Problem:
    if response.status_code == 200:
        response_200 = DigestHistory.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DigestHistory | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 5,

) -> Response[DigestHistory | Problem]:
    """ What happened since the last issue: posts that went in or were left out, issues, and who turned it
    on

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        limit (int | Unset):  Default: 5.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestHistory | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 5,

) -> DigestHistory | Problem | None:
    """ What happened since the last issue: posts that went in or were left out, issues, and who turned it
    on

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        limit (int | Unset):  Default: 5.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestHistory | Problem
     """


    return sync_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 5,

) -> Response[DigestHistory | Problem]:
    """ What happened since the last issue: posts that went in or were left out, issues, and who turned it
    on

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        limit (int | Unset):  Default: 5.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestHistory | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 5,

) -> DigestHistory | Problem | None:
    """ What happened since the last issue: posts that went in or were left out, issues, and who turned it
    on

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        limit (int | Unset):  Default: 5.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestHistory | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,
limit=limit,

    )).parsed
