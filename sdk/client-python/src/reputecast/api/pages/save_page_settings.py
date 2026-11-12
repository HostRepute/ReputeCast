from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.page_detail import PageDetail
from ...models.page_settings_input import PageSettingsInput
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    page: UUID,
    *,
    body: PageSettingsInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/pages/{page}/settings".format(workspace=quote(str(workspace), safe=""),page=quote(str(page), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
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
    *,
    client: AuthenticatedClient,
    body: PageSettingsInput,

) -> Response[PageDetail | Problem]:
    """ Save the Page tab: title, address, visibility, header and navigation, SEO

     Visibility and navigation need website:manage.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        body (PageSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PageDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
body=body,

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
    body: PageSettingsInput,

) -> PageDetail | Problem | None:
    """ Save the Page tab: title, address, visibility, header and navigation, SEO

     Visibility and navigation need website:manage.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        body (PageSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PageDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
page=page,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    page: UUID,
    *,
    client: AuthenticatedClient,
    body: PageSettingsInput,

) -> Response[PageDetail | Problem]:
    """ Save the Page tab: title, address, visibility, header and navigation, SEO

     Visibility and navigation need website:manage.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        body (PageSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PageDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
body=body,

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
    body: PageSettingsInput,

) -> PageDetail | Problem | None:
    """ Save the Page tab: title, address, visibility, header and navigation, SEO

     Visibility and navigation need website:manage.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (UUID): The page's id
        body (PageSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PageDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
page=page,
client=client,
body=body,

    )).parsed
