from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_sponsor_overview_range import GetSponsorOverviewRange
from ...models.problem import Problem
from ...models.sponsor_overview import SponsorOverview
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    slot: str | Unset = UNSET,
    range_: GetSponsorOverviewRange | Unset = GetSponsorOverviewRange.AROUND,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["slot"] = slot

    json_range_: str | Unset = UNSET
    if not isinstance(range_, Unset):
        json_range_ = range_.value

    params["range"] = json_range_


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/sponsors/overview".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SponsorOverview:
    if response.status_code == 200:
        response_200 = SponsorOverview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SponsorOverview]:
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
    slot: str | Unset = UNSET,
    range_: GetSponsorOverviewRange | Unset = GetSponsorOverviewRange.AROUND,

) -> Response[Problem | SponsorOverview]:
    """ Earn › Sponsors: income, a slot's issues with their bookings, the library and what needs a person

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        slot (str | Unset): A slot's id; the first slot when absent.
        range_ (GetSponsorOverviewRange | Unset):  Default: GetSponsorOverviewRange.AROUND.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
slot=slot,
range_=range_,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    slot: str | Unset = UNSET,
    range_: GetSponsorOverviewRange | Unset = GetSponsorOverviewRange.AROUND,

) -> Problem | SponsorOverview | None:
    """ Earn › Sponsors: income, a slot's issues with their bookings, the library and what needs a person

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        slot (str | Unset): A slot's id; the first slot when absent.
        range_ (GetSponsorOverviewRange | Unset):  Default: GetSponsorOverviewRange.AROUND.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorOverview
     """


    return sync_detailed(
        workspace=workspace,
client=client,
slot=slot,
range_=range_,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    slot: str | Unset = UNSET,
    range_: GetSponsorOverviewRange | Unset = GetSponsorOverviewRange.AROUND,

) -> Response[Problem | SponsorOverview]:
    """ Earn › Sponsors: income, a slot's issues with their bookings, the library and what needs a person

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        slot (str | Unset): A slot's id; the first slot when absent.
        range_ (GetSponsorOverviewRange | Unset):  Default: GetSponsorOverviewRange.AROUND.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
slot=slot,
range_=range_,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    slot: str | Unset = UNSET,
    range_: GetSponsorOverviewRange | Unset = GetSponsorOverviewRange.AROUND,

) -> Problem | SponsorOverview | None:
    """ Earn › Sponsors: income, a slot's issues with their bookings, the library and what needs a person

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        slot (str | Unset): A slot's id; the first slot when absent.
        range_ (GetSponsorOverviewRange | Unset):  Default: GetSponsorOverviewRange.AROUND.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorOverview
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
slot=slot,
range_=range_,

    )).parsed
