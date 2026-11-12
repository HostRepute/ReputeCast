from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_product_orders_period import ListProductOrdersPeriod
from ...models.list_product_orders_status import ListProductOrdersStatus
from ...models.problem import Problem
from ...models.product_order_list import ProductOrderList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,
    product: str | Unset = UNSET,
    status: ListProductOrdersStatus | Unset = UNSET,
    period: ListProductOrdersPeriod | Unset = ListProductOrdersPeriod.VALUE_1,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q

    params["product"] = product

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period

    params["page"] = page

    params["pageSize"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/products/orders".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProductOrderList:
    if response.status_code == 200:
        response_200 = ProductOrderList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProductOrderList]:
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
    q: str | Unset = UNSET,
    product: str | Unset = UNSET,
    status: ListProductOrdersStatus | Unset = UNSET,
    period: ListProductOrdersPeriod | Unset = ListProductOrdersPeriod.VALUE_1,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> Response[Problem | ProductOrderList]:
    """ Orders, newest first, with their download links

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): An order number or part of the buyer's email
        product (str | Unset): A product's id
        status (ListProductOrdersStatus | Unset):
        period (ListProductOrdersPeriod | Unset):  Default: ListProductOrdersPeriod.VALUE_1.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductOrderList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
product=product,
status=status,
period=period,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    product: str | Unset = UNSET,
    status: ListProductOrdersStatus | Unset = UNSET,
    period: ListProductOrdersPeriod | Unset = ListProductOrdersPeriod.VALUE_1,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> Problem | ProductOrderList | None:
    """ Orders, newest first, with their download links

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): An order number or part of the buyer's email
        product (str | Unset): A product's id
        status (ListProductOrdersStatus | Unset):
        period (ListProductOrdersPeriod | Unset):  Default: ListProductOrdersPeriod.VALUE_1.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductOrderList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,
product=product,
status=status,
period=period,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    product: str | Unset = UNSET,
    status: ListProductOrdersStatus | Unset = UNSET,
    period: ListProductOrdersPeriod | Unset = ListProductOrdersPeriod.VALUE_1,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> Response[Problem | ProductOrderList]:
    """ Orders, newest first, with their download links

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): An order number or part of the buyer's email
        product (str | Unset): A product's id
        status (ListProductOrdersStatus | Unset):
        period (ListProductOrdersPeriod | Unset):  Default: ListProductOrdersPeriod.VALUE_1.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductOrderList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
product=product,
status=status,
period=period,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    product: str | Unset = UNSET,
    status: ListProductOrdersStatus | Unset = UNSET,
    period: ListProductOrdersPeriod | Unset = ListProductOrdersPeriod.VALUE_1,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> Problem | ProductOrderList | None:
    """ Orders, newest first, with their download links

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): An order number or part of the buyer's email
        product (str | Unset): A product's id
        status (ListProductOrdersStatus | Unset):
        period (ListProductOrdersPeriod | Unset):  Default: ListProductOrdersPeriod.VALUE_1.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductOrderList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,
product=product,
status=status,
period=period,
page=page,
page_size=page_size,

    )).parsed
