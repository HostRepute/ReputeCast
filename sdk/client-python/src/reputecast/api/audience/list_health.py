from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_health import ListHealth
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/audience/health".format(workspace=quote(str(workspace), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ListHealth | Problem:
    if response.status_code == 200:
        response_200 = ListHealth.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ListHealth | Problem]:
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

) -> Response[ListHealth | Problem]:
    """ Tiers, tier trend, sunset policy and health per list

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ListHealth | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> ListHealth | Problem | None:
    """ Tiers, tier trend, sunset policy and health per list

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ListHealth | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> Response[ListHealth | Problem]:
    """ Tiers, tier trend, sunset policy and health per list

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ListHealth | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> ListHealth | Problem | None:
    """ Tiers, tier trend, sunset policy and health per list

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ListHealth | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,

    )).parsed
