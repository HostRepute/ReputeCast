from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_revenue_overview_period import GetRevenueOverviewPeriod
from ...models.problem import Problem
from ...models.revenue_overview import RevenueOverview
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    period: GetRevenueOverviewPeriod | Unset = GetRevenueOverviewPeriod.VALUE_0,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/earn/revenue".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | RevenueOverview:
    if response.status_code == 200:
        response_200 = RevenueOverview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | RevenueOverview]:
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
    period: GetRevenueOverviewPeriod | Unset = GetRevenueOverviewPeriod.VALUE_0,

) -> Response[Problem | RevenueOverview]:
    """ Earn › Overview: MRR and its movement, paid members, trials, conversions, cancellations and
    attribution

     MRR is the monthly value of every Active and Past due subscription, after discounts, before tax,
    live mode only; a yearly price counts as a twelfth.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetRevenueOverviewPeriod | Unset): The last 30 days, 90 days or 12 months Default:
            GetRevenueOverviewPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RevenueOverview]
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
    period: GetRevenueOverviewPeriod | Unset = GetRevenueOverviewPeriod.VALUE_0,

) -> Problem | RevenueOverview | None:
    """ Earn › Overview: MRR and its movement, paid members, trials, conversions, cancellations and
    attribution

     MRR is the monthly value of every Active and Past due subscription, after discounts, before tax,
    live mode only; a yearly price counts as a twelfth.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetRevenueOverviewPeriod | Unset): The last 30 days, 90 days or 12 months Default:
            GetRevenueOverviewPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RevenueOverview
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
    period: GetRevenueOverviewPeriod | Unset = GetRevenueOverviewPeriod.VALUE_0,

) -> Response[Problem | RevenueOverview]:
    """ Earn › Overview: MRR and its movement, paid members, trials, conversions, cancellations and
    attribution

     MRR is the monthly value of every Active and Past due subscription, after discounts, before tax,
    live mode only; a yearly price counts as a twelfth.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetRevenueOverviewPeriod | Unset): The last 30 days, 90 days or 12 months Default:
            GetRevenueOverviewPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RevenueOverview]
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
    period: GetRevenueOverviewPeriod | Unset = GetRevenueOverviewPeriod.VALUE_0,

) -> Problem | RevenueOverview | None:
    """ Earn › Overview: MRR and its movement, paid members, trials, conversions, cancellations and
    attribution

     MRR is the monthly value of every Active and Past due subscription, after discounts, before tax,
    live mode only; a yearly price counts as a twelfth.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetRevenueOverviewPeriod | Unset): The last 30 days, 90 days or 12 months Default:
            GetRevenueOverviewPeriod.VALUE_0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RevenueOverview
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
period=period,

    )).parsed
