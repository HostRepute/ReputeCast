from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.placement_copy_detail import PlacementCopyDetail
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    test: UUID,
    copy: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/placement/tests/{test}/copies/{copy}".format(workspace=quote(str(workspace), safe=""),test=quote(str(test), safe=""),copy=quote(str(copy), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PlacementCopyDetail | Problem:
    if response.status_code == 200:
        response_200 = PlacementCopyDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PlacementCopyDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    test: UUID,
    copy: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PlacementCopyDetail | Problem]:
    """ One copy: where it landed, its headers and the provider's events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        test (UUID):
        copy (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PlacementCopyDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
test=test,
copy=copy,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    test: UUID,
    copy: UUID,
    *,
    client: AuthenticatedClient,

) -> PlacementCopyDetail | Problem | None:
    """ One copy: where it landed, its headers and the provider's events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        test (UUID):
        copy (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PlacementCopyDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
test=test,
copy=copy,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    test: UUID,
    copy: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PlacementCopyDetail | Problem]:
    """ One copy: where it landed, its headers and the provider's events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        test (UUID):
        copy (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PlacementCopyDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
test=test,
copy=copy,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    test: UUID,
    copy: UUID,
    *,
    client: AuthenticatedClient,

) -> PlacementCopyDetail | Problem | None:
    """ One copy: where it landed, its headers and the provider's events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        test (UUID):
        copy (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PlacementCopyDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
test=test,
copy=copy,
client=client,

    )).parsed
