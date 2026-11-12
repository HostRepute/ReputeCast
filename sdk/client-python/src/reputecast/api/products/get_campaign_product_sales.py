from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.campaign_product_sales import CampaignProductSales
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign}/product-sales".format(workspace=quote(str(workspace), safe=""),campaign=quote(str(campaign), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CampaignProductSales | Problem:
    if response.status_code == 200:
        response_200 = CampaignProductSales.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CampaignProductSales | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[CampaignProductSales | Problem]:
    """ Product sales credited to a campaign (its report's Revenue rows)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignProductSales | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign=campaign,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign: UUID,
    *,
    client: AuthenticatedClient,

) -> CampaignProductSales | Problem | None:
    """ Product sales credited to a campaign (its report's Revenue rows)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignProductSales | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign=campaign,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[CampaignProductSales | Problem]:
    """ Product sales credited to a campaign (its report's Revenue rows)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignProductSales | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign=campaign,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign: UUID,
    *,
    client: AuthenticatedClient,

) -> CampaignProductSales | Problem | None:
    """ Product sales credited to a campaign (its report's Revenue rows)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignProductSales | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign=campaign,
client=client,

    )).parsed
