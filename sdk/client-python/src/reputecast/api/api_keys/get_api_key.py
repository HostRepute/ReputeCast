from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.api_key_view import APIKeyView
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    key_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/api-keys/{key_id}".format(workspace=quote(str(workspace), safe=""),key_id=quote(str(key_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> APIKeyView | Problem:
    if response.status_code == 200:
        response_200 = APIKeyView.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[APIKeyView | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    key_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[APIKeyView | Problem]:
    """ One API key with the calls rejected since it expired

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        key_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[APIKeyView | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
key_id=key_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    key_id: UUID,
    *,
    client: AuthenticatedClient,

) -> APIKeyView | Problem | None:
    """ One API key with the calls rejected since it expired

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        key_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        APIKeyView | Problem
     """


    return sync_detailed(
        workspace=workspace,
key_id=key_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    key_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[APIKeyView | Problem]:
    """ One API key with the calls rejected since it expired

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        key_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[APIKeyView | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
key_id=key_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    key_id: UUID,
    *,
    client: AuthenticatedClient,

) -> APIKeyView | Problem | None:
    """ One API key with the calls rejected since it expired

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        key_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        APIKeyView | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
key_id=key_id,
client=client,

    )).parsed
