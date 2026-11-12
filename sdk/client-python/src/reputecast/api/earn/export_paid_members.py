from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.export_paid_members_period import ExportPaidMembersPeriod
from ...models.export_paid_members_since import ExportPaidMembersSince
from ...models.export_paid_members_source import ExportPaidMembersSource
from ...models.export_paid_members_status import ExportPaidMembersStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    workspace: str,
    *,
    status: ExportPaidMembersStatus | Unset = ExportPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ExportPaidMembersPeriod | Unset = UNSET,
    source: ExportPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ExportPaidMembersSince | Unset = UNSET,
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
        "url": "/api/v1/workspaces/{workspace}/earn/members/export".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | str:
    if response.status_code == 200:
        response_200 = cast(str, response.json())
        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | str]:
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
    status: ExportPaidMembersStatus | Unset = ExportPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ExportPaidMembersPeriod | Unset = UNSET,
    source: ExportPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ExportPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Response[Problem | str]:
    """ Export the paid members of a filter as CSV

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportPaidMembersStatus | Unset):  Default: ExportPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ExportPaidMembersPeriod | Unset):
        source (ExportPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ExportPaidMembersSince | Unset):
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
        Response[Problem | str]
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
    status: ExportPaidMembersStatus | Unset = ExportPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ExportPaidMembersPeriod | Unset = UNSET,
    source: ExportPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ExportPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Problem | str | None:
    """ Export the paid members of a filter as CSV

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportPaidMembersStatus | Unset):  Default: ExportPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ExportPaidMembersPeriod | Unset):
        source (ExportPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ExportPaidMembersSince | Unset):
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
        Problem | str
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
    status: ExportPaidMembersStatus | Unset = ExportPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ExportPaidMembersPeriod | Unset = UNSET,
    source: ExportPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ExportPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Response[Problem | str]:
    """ Export the paid members of a filter as CSV

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportPaidMembersStatus | Unset):  Default: ExportPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ExportPaidMembersPeriod | Unset):
        source (ExportPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ExportPaidMembersSince | Unset):
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
        Response[Problem | str]
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
    status: ExportPaidMembersStatus | Unset = ExportPaidMembersStatus.WITH_ACCESS,
    q: str | Unset = UNSET,
    tier: str | Unset = UNSET,
    period: ExportPaidMembersPeriod | Unset = UNSET,
    source: ExportPaidMembersSource | Unset = UNSET,
    offer: str | Unset = UNSET,
    since: ExportPaidMembersSince | Unset = UNSET,
    campaign: str | Unset = UNSET,
    before: datetime.date | Unset = UNSET,
    test: bool | Unset = UNSET,
    all_: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Problem | str | None:
    """ Export the paid members of a filter as CSV

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportPaidMembersStatus | Unset):  Default: ExportPaidMembersStatus.WITH_ACCESS.
        q (str | Unset): Name, email, billing email or Stripe customer ID
        tier (str | Unset): Tier id
        period (ExportPaidMembersPeriod | Unset):
        source (ExportPaidMembersSource | Unset):
        offer (str | Unset): An offer's label (FALL26)
        since (ExportPaidMembersSince | Unset):
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
        Problem | str
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
