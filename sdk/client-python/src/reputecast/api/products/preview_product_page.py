from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.public_page_preview import PublicPagePreview
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    product: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/products/{product}/preview".format(workspace=quote(str(workspace), safe=""),product=quote(str(product), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | PublicPagePreview:
    if response.status_code == 200:
        response_200 = PublicPagePreview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | PublicPagePreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a product's default page (used when it has no sales page)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
product=product,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | PublicPagePreview | None:
    """ Preview a product's default page (used when it has no sales page)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPagePreview
     """


    return sync_detailed(
        workspace=workspace,
product=product,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a product's default page (used when it has no sales page)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
product=product,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | PublicPagePreview | None:
    """ Preview a product's default page (used when it has no sales page)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPagePreview
     """


    return (await asyncio_detailed(
        workspace=workspace,
product=product,
client=client,

    )).parsed
