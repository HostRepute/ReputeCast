from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.digest_check import DigestCheck
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    digest_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/digests/{digest_id}/check".format(workspace=quote(str(workspace), safe=""),digest_id=quote(str(digest_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DigestCheck | Problem:
    if response.status_code == 200:
        response_200 = DigestCheck.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DigestCheck | Problem]:
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

) -> Response[DigestCheck | Problem]:
    """ Run Send Readiness on the next issue as it would be built now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestCheck | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,

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

) -> DigestCheck | Problem | None:
    """ Run Send Readiness on the next issue as it would be built now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestCheck | Problem
     """


    return sync_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[DigestCheck | Problem]:
    """ Run Send Readiness on the next issue as it would be built now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestCheck | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,

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

) -> DigestCheck | Problem | None:
    """ Run Send Readiness on the next issue as it would be built now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestCheck | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,

    )).parsed
