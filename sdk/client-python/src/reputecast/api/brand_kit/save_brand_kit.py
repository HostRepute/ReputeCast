from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.brand_kit import BrandKit
from ...models.brand_kit_input import BrandKitInput
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: BrandKitInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/brand-kit".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BrandKit | Problem:
    if response.status_code == 200:
        response_200 = BrandKit.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BrandKit | Problem]:
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
    body: BrandKitInput,

) -> Response[BrandKit | Problem]:
    """ Save the brand kit

     New emails, templates and blocks start from it; emails already written keep their own styles. The
    first save creates the synced Header and Footer; a new logo reaches the synced Header, and the
    drafts and scheduled emails holding it when the caller also holds campaigns:edit.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (BrandKitInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BrandKit | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: BrandKitInput,

) -> BrandKit | Problem | None:
    """ Save the brand kit

     New emails, templates and blocks start from it; emails already written keep their own styles. The
    first save creates the synced Header and Footer; a new logo reaches the synced Header, and the
    drafts and scheduled emails holding it when the caller also holds campaigns:edit.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (BrandKitInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BrandKit | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: BrandKitInput,

) -> Response[BrandKit | Problem]:
    """ Save the brand kit

     New emails, templates and blocks start from it; emails already written keep their own styles. The
    first save creates the synced Header and Footer; a new logo reaches the synced Header, and the
    drafts and scheduled emails holding it when the caller also holds campaigns:edit.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (BrandKitInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BrandKit | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: BrandKitInput,

) -> BrandKit | Problem | None:
    """ Save the brand kit

     New emails, templates and blocks start from it; emails already written keep their own styles. The
    first save creates the synced Header and Footer; a new logo reaches the synced Header, and the
    drafts and scheduled emails holding it when the caller also holds campaigns:edit.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (BrandKitInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BrandKit | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
