from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.earn_member_list import EarnMemberList
from ...models.list_paid_members_period import ListPaidMembersPeriod
from ...models.list_paid_members_since import ListPaidMembersSince
from ...models.list_paid_members_source import ListPaidMembersSource
from ...models.list_paid_members_status import ListPaidMembersStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    workspace: str,
    *,
    status: ListPaidMembersStatus | Unset = ListPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ListPaidMembersPeriod | Unset = UNSET,
    source: ListPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ListPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["q"] = q

    params["tier"] = tier

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period

    json_source: str | Unset = UNSET
    if not isinstance(source, Unset):
        json_source = source.value

    params["source"] = json_source

    params["offer"] = offer

    json_since: str | Unset = UNSET
    if not isinstance(since, Unset):
        json_since = since.value

    params["since"] = json_since

    params["campaign"] = campaign

    json_before: str | Unset = UNSET
    if not isinstance(before, Unset):
        json_before = before.isoformat()
    params["before"] = json_before

    params["test"] = test

    params["all"] = all_

    params["page"] = page

    params["pageSize"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/earn/members".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EarnMemberList | Problem:
    if response.status_code == 200:
        response_200 = EarnMemberList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EarnMemberList | Problem]:
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
    status: ListPaidMembersStatus | Unset = ListPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ListPaidMembersPeriod | Unset = UNSET,
    source: ListPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ListPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Response[EarnMemberList | Problem]:
    """ Paid members, with their counts by status

     Amounts are in minor units, before tax. Test paid members show only with test=true.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListPaidMembersStatus | Unset):  Default: ListPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ListPaidMembersPeriod | Unset):
        source (ListPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ListPaidMembersSince | Unset):
        campaign (str | Unset): Campaign id: paid members credited to it
        before (datetime.date | Unset): Paid members since before this date
        test (bool | Unset):
        all_ (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EarnMemberList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
q=q,
tier=tier,
period=period,
source=source,
offer=offer,
since=since,
campaign=campaign,
before=before,
test=test,
all_=all_,
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
    status: ListPaidMembersStatus | Unset = ListPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ListPaidMembersPeriod | Unset = UNSET,
    source: ListPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ListPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> EarnMemberList | Problem | None:
    """ Paid members, with their counts by status

     Amounts are in minor units, before tax. Test paid members show only with test=true.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListPaidMembersStatus | Unset):  Default: ListPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ListPaidMembersPeriod | Unset):
        source (ListPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ListPaidMembersSince | Unset):
        campaign (str | Unset): Campaign id: paid members credited to it
        before (datetime.date | Unset): Paid members since before this date
        test (bool | Unset):
        all_ (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EarnMemberList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,
q=q,
tier=tier,
period=period,
source=source,
offer=offer,
since=since,
campaign=campaign,
before=before,
test=test,
all_=all_,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListPaidMembersStatus | Unset = ListPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ListPaidMembersPeriod | Unset = UNSET,
    source: ListPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ListPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Response[EarnMemberList | Problem]:
    """ Paid members, with their counts by status

     Amounts are in minor units, before tax. Test paid members show only with test=true.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListPaidMembersStatus | Unset):  Default: ListPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ListPaidMembersPeriod | Unset):
        source (ListPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ListPaidMembersSince | Unset):
        campaign (str | Unset): Campaign id: paid members credited to it
        before (datetime.date | Unset): Paid members since before this date
        test (bool | Unset):
        all_ (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EarnMemberList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
q=q,
tier=tier,
period=period,
source=source,
offer=offer,
since=since,
campaign=campaign,
before=before,
test=test,
all_=all_,
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
    status: ListPaidMembersStatus | Unset = ListPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ListPaidMembersPeriod | Unset = UNSET,
    source: ListPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ListPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> EarnMemberList | Problem | None:
    """ Paid members, with their counts by status

     Amounts are in minor units, before tax. Test paid members show only with test=true.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListPaidMembersStatus | Unset):  Default: ListPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ListPaidMembersPeriod | Unset):
        source (ListPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ListPaidMembersSince | Unset):
        campaign (str | Unset): Campaign id: paid members credited to it
        before (datetime.date | Unset): Paid members since before this date
        test (bool | Unset):
        all_ (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EarnMemberList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,
q=q,
tier=tier,
period=period,
source=source,
offer=offer,
since=since,
campaign=campaign,
before=before,
test=test,
all_=all_,
page=page,
page_size=page_size,

    )).parsed
