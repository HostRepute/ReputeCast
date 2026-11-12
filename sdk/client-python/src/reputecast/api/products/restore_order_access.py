from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.product_order_detail import ProductOrderDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    order: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/products/orders/{order}/restore".format(workspace=quote(str(workspace), safe=""),order=quote(str(order), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProductOrderDetail:
    if response.status_code == 200:
        response_200 = ProductOrderDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProductOrderDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    order: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ProductOrderDetail]:
    """ Let an order's downloads work again

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        order (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductOrderDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
order=order,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    order: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ProductOrderDetail | None:
    """ Let an order's downloads work again

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        order (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductOrderDetail
     """


    return sync_detailed(
        workspace=workspace,
order=order,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    order: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ProductOrderDetail]:
    """ Let an order's downloads work again

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        order (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductOrderDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
order=order,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    order: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ProductOrderDetail | None:
    """ Let an order's downloads work again

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        order (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductOrderDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
order=order,
client=client,

    )).parsed
