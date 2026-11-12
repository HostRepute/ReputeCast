from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_products_period import ListProductsPeriod
from ...models.problem import Problem
from ...models.products_overview import ProductsOverview
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    period: ListProductsPeriod | Unset = ListProductsPeriod.VALUE_0,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/products".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProductsOverview:
    if response.status_code == 200:
        response_200 = ProductsOverview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProductsOverview]:
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
    period: ListProductsPeriod | Unset = ListProductsPeriod.VALUE_0,

) -> Response[Problem | ProductsOverview]:
    """ Products with their numbers, where sales came from and the payment connection

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListProductsPeriod | Unset):  Default: ListProductsPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductsOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: ListProductsPeriod | Unset = ListProductsPeriod.VALUE_0,

) -> Problem | ProductsOverview | None:
    """ Products with their numbers, where sales came from and the payment connection

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListProductsPeriod | Unset):  Default: ListProductsPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductsOverview
     """


    return sync_detailed(
        workspace=workspace,
client=client,
period=period,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: ListProductsPeriod | Unset = ListProductsPeriod.VALUE_0,

) -> Response[Problem | ProductsOverview]:
    """ Products with their numbers, where sales came from and the payment connection

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListProductsPeriod | Unset):  Default: ListProductsPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductsOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: ListProductsPeriod | Unset = ListProductsPeriod.VALUE_0,

) -> Problem | ProductsOverview | None:
    """ Products with their numbers, where sales came from and the payment connection

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListProductsPeriod | Unset):  Default: ListProductsPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductsOverview
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
period=period,

    )).parsed
