from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.coupon_pool import CouponPool
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    pool: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/coupons/pools/{pool}".format(workspace=quote(str(workspace), safe=""),pool=quote(str(pool), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CouponPool | Problem:
    if response.status_code == 200:
        response_200 = CouponPool.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CouponPool | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    pool: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[CouponPool | Problem]:
    """ A discount code pool: numbers, settings, the latest batch and what uses it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        pool (UUID): Pool id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CouponPool | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
pool=pool,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    pool: UUID,
    *,
    client: AuthenticatedClient,

) -> CouponPool | Problem | None:
    """ A discount code pool: numbers, settings, the latest batch and what uses it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        pool (UUID): Pool id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CouponPool | Problem
     """


    return sync_detailed(
        workspace=workspace,
pool=pool,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    pool: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[CouponPool | Problem]:
    """ A discount code pool: numbers, settings, the latest batch and what uses it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        pool (UUID): Pool id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CouponPool | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
pool=pool,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    pool: UUID,
    *,
    client: AuthenticatedClient,

) -> CouponPool | Problem | None:
    """ A discount code pool: numbers, settings, the latest batch and what uses it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        pool (UUID): Pool id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CouponPool | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
pool=pool,
client=client,

    )).parsed
