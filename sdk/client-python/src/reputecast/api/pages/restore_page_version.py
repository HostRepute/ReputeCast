from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.page_detail import PageDetail
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    page: UUID,
    version: int,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/pages/{page}/versions/{version}/restore".format(workspace=quote(str(workspace), safe=""),page=quote(str(page), safe=""),version=quote(str(version), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PageDetail | Problem:
    if response.status_code == 200:
        response_200 = PageDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PageDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    page: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> Response[PageDetail | Problem]:
    """ Make an earlier version the draft again, as a new version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PageDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
version=version,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    page: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> PageDetail | Problem | None:
    """ Make an earlier version the draft again, as a new version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PageDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
page=page,
version=version,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    page: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> Response[PageDetail | Problem]:
    """ Make an earlier version the draft again, as a new version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PageDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
version=version,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    page: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> PageDetail | Problem | None:
    """ Make an earlier version the draft again, as a new version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PageDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
page=page,
version=version,
client=client,

    )).parsed
