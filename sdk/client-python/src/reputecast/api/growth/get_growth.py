from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_growth_period import GetGrowthPeriod
from ...models.growth_page import GrowthPage
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    period: GetGrowthPeriod | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    imports: bool | Unset = UNSET,
    cohort_source: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period

    params["from"] = from_

    params["to"] = to

    params["list"] = list_

    params["imports"] = imports

    params["cohortSource"] = cohort_source


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/growth".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> GrowthPage | Problem:
    if response.status_code == 200:
        response_200 = GrowthPage.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[GrowthPage | Problem]:
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
    period: GetGrowthPeriod | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    imports: bool | Unset = UNSET,
    cohort_source: str | Unset = UNSET,

) -> Response[GrowthPage | Problem]:
    """ Joins and departures by source, quality by source, UTM and referring pages, and cohort retention

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetGrowthPeriod | Unset): 30d (default), 90d, 12m or custom
        from_ (str | Unset): First day of a custom period (YYYY-MM-DD)
        to (str | Unset): Last day of a custom period
        list_ (str | Unset): A list id: count its members instead of contacts
        imports (bool | Unset):
        cohort_source (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GrowthPage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
from_=from_,
to=to,
list_=list_,
imports=imports,
cohort_source=cohort_source,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: GetGrowthPeriod | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    imports: bool | Unset = UNSET,
    cohort_source: str | Unset = UNSET,

) -> GrowthPage | Problem | None:
    """ Joins and departures by source, quality by source, UTM and referring pages, and cohort retention

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetGrowthPeriod | Unset): 30d (default), 90d, 12m or custom
        from_ (str | Unset): First day of a custom period (YYYY-MM-DD)
        to (str | Unset): Last day of a custom period
        list_ (str | Unset): A list id: count its members instead of contacts
        imports (bool | Unset):
        cohort_source (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GrowthPage | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
period=period,
from_=from_,
to=to,
list_=list_,
imports=imports,
cohort_source=cohort_source,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: GetGrowthPeriod | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    imports: bool | Unset = UNSET,
    cohort_source: str | Unset = UNSET,

) -> Response[GrowthPage | Problem]:
    """ Joins and departures by source, quality by source, UTM and referring pages, and cohort retention

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetGrowthPeriod | Unset): 30d (default), 90d, 12m or custom
        from_ (str | Unset): First day of a custom period (YYYY-MM-DD)
        to (str | Unset): Last day of a custom period
        list_ (str | Unset): A list id: count its members instead of contacts
        imports (bool | Unset):
        cohort_source (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GrowthPage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
from_=from_,
to=to,
list_=list_,
imports=imports,
cohort_source=cohort_source,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: GetGrowthPeriod | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    imports: bool | Unset = UNSET,
    cohort_source: str | Unset = UNSET,

) -> GrowthPage | Problem | None:
    """ Joins and departures by source, quality by source, UTM and referring pages, and cohort retention

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetGrowthPeriod | Unset): 30d (default), 90d, 12m or custom
        from_ (str | Unset): First day of a custom period (YYYY-MM-DD)
        to (str | Unset): Last day of a custom period
        list_ (str | Unset): A list id: count its members instead of contacts
        imports (bool | Unset):
        cohort_source (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GrowthPage | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
period=period,
from_=from_,
to=to,
list_=list_,
imports=imports,
cohort_source=cohort_source,

    )).parsed
