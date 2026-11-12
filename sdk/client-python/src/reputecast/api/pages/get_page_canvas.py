from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.page_canvas import PageCanvas
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    page: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/pages/{page}/canvas".format(workspace=quote(str(workspace), safe=""),page=quote(str(page), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PageCanvas | Problem:
    if response.status_code == 200:
        response_200 = PageCanvas.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PageCanvas | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    page: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PageCanvas | Problem]:
    """ The site's stylesheets, header and footer for the page builder's canvas

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PageCanvas | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    page: UUID,
    *,
    client: AuthenticatedClient,

) -> PageCanvas | Problem | None:
    """ The site's stylesheets, header and footer for the page builder's canvas

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PageCanvas | Problem
     """


    return sync_detailed(
        workspace=workspace,
page=page,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    page: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PageCanvas | Problem]:
    """ The site's stylesheets, header and footer for the page builder's canvas

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PageCanvas | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    page: UUID,
    *,
    client: AuthenticatedClient,

) -> PageCanvas | Problem | None:
    """ The site's stylesheets, header and footer for the page builder's canvas

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PageCanvas | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
page=page,
client=client,

    )).parsed
