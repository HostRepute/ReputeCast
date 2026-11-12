from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_referral_results_grouping import GetReferralResultsGrouping
from ...models.get_referral_results_period import GetReferralResultsPeriod
from ...models.problem import Problem
from ...models.referral_results import ReferralResults
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    program: UUID,
    *,
    period: GetReferralResultsPeriod | Unset = GetReferralResultsPeriod.ALL,
    grouping: GetReferralResultsGrouping | Unset = GetReferralResultsGrouping.WEEK,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period

    json_grouping: str | Unset = UNSET
    if not isinstance(grouping, Unset):
        json_grouping = grouping.value

    params["grouping"] = json_grouping


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/referrals/{program}/results".format(workspace=quote(str(workspace), safe=""),program=quote(str(program), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReferralResults:
    if response.status_code == 200:
        response_200 = ReferralResults.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReferralResults]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    period: GetReferralResultsPeriod | Unset = GetReferralResultsPeriod.ALL,
    grouping: GetReferralResultsGrouping | Unset = GetReferralResultsGrouping.WEEK,

) -> Response[Problem | ReferralResults]:
    """ A program's results: the funnel, confirmed referrals over time, channels, top referrers, rewards and
    quality

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        period (GetReferralResultsPeriod | Unset):  Default: GetReferralResultsPeriod.ALL.
        grouping (GetReferralResultsGrouping | Unset):  Default: GetReferralResultsGrouping.WEEK.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralResults]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
period=period,
grouping=grouping,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    period: GetReferralResultsPeriod | Unset = GetReferralResultsPeriod.ALL,
    grouping: GetReferralResultsGrouping | Unset = GetReferralResultsGrouping.WEEK,

) -> Problem | ReferralResults | None:
    """ A program's results: the funnel, confirmed referrals over time, channels, top referrers, rewards and
    quality

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        period (GetReferralResultsPeriod | Unset):  Default: GetReferralResultsPeriod.ALL.
        grouping (GetReferralResultsGrouping | Unset):  Default: GetReferralResultsGrouping.WEEK.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralResults
     """


    return sync_detailed(
        workspace=workspace,
program=program,
client=client,
period=period,
grouping=grouping,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    period: GetReferralResultsPeriod | Unset = GetReferralResultsPeriod.ALL,
    grouping: GetReferralResultsGrouping | Unset = GetReferralResultsGrouping.WEEK,

) -> Response[Problem | ReferralResults]:
    """ A program's results: the funnel, confirmed referrals over time, channels, top referrers, rewards and
    quality

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        period (GetReferralResultsPeriod | Unset):  Default: GetReferralResultsPeriod.ALL.
        grouping (GetReferralResultsGrouping | Unset):  Default: GetReferralResultsGrouping.WEEK.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralResults]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
period=period,
grouping=grouping,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    period: GetReferralResultsPeriod | Unset = GetReferralResultsPeriod.ALL,
    grouping: GetReferralResultsGrouping | Unset = GetReferralResultsGrouping.WEEK,

) -> Problem | ReferralResults | None:
    """ A program's results: the funnel, confirmed referrals over time, channels, top referrers, rewards and
    quality

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        period (GetReferralResultsPeriod | Unset):  Default: GetReferralResultsPeriod.ALL.
        grouping (GetReferralResultsGrouping | Unset):  Default: GetReferralResultsGrouping.WEEK.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralResults
     """


    return (await asyncio_detailed(
        workspace=workspace,
program=program,
client=client,
period=period,
grouping=grouping,

    )).parsed
