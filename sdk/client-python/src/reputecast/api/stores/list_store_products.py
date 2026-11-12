from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.commerce_product_list import CommerceProductList
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    collection: str | Unset = UNSET,
    search: str | Unset = UNSET,
    cursor: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["collection"] = collection

    params["search"] = search

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/stores/{id}/products".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CommerceProductList | Problem:
    if response.status_code == 200:
        response_200 = CommerceProductList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CommerceProductList | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    collection: str | Unset = UNSET,
    search: str | Unset = UNSET,
    cursor: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[CommerceProductList | Problem]:
    """ A store's products, newest first, with their prices and availability now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The store's id
        collection (str | Unset):
        search (str | Unset):
        cursor (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CommerceProductList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
collection=collection,
search=search,
cursor=cursor,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    collection: str | Unset = UNSET,
    search: str | Unset = UNSET,
    cursor: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> CommerceProductList | Problem | None:
    """ A store's products, newest first, with their prices and availability now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The store's id
        collection (str | Unset):
        search (str | Unset):
        cursor (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CommerceProductList | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
collection=collection,
search=search,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    collection: str | Unset = UNSET,
    search: str | Unset = UNSET,
    cursor: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[CommerceProductList | Problem]:
    """ A store's products, newest first, with their prices and availability now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The store's id
        collection (str | Unset):
        search (str | Unset):
        cursor (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CommerceProductList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
collection=collection,
search=search,
cursor=cursor,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    collection: str | Unset = UNSET,
    search: str | Unset = UNSET,
    cursor: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> CommerceProductList | Problem | None:
    """ A store's products, newest first, with their prices and availability now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The store's id
        collection (str | Unset):
        search (str | Unset):
        cursor (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CommerceProductList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
collection=collection,
search=search,
cursor=cursor,
limit=limit,

    )).parsed
