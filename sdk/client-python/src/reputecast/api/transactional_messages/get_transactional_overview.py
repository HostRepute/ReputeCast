from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_transactional_overview_period import GetTransactionalOverviewPeriod
from ...models.problem import Problem
from ...models.transactional_overview import TransactionalOverview
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    period: GetTransactionalOverviewPeriod | Unset = GetTransactionalOverviewPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period

    params["includeTest"] = include_test


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/transactional/overview".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalOverview:
    if response.status_code == 200:
        response_200 = TransactionalOverview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalOverview]:
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
    period: GetTransactionalOverviewPeriod | Unset = GetTransactionalOverviewPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Response[Problem | TransactionalOverview]:
    """ The period's numbers, the route, alerts, incidents and the keys that can send

     The Templates tab's header and side cards, and the Overview's Transactional card.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetTransactionalOverviewPeriod | Unset):  Default:
            GetTransactionalOverviewPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
include_test=include_test,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: GetTransactionalOverviewPeriod | Unset = GetTransactionalOverviewPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Problem | TransactionalOverview | None:
    """ The period's numbers, the route, alerts, incidents and the keys that can send

     The Templates tab's header and side cards, and the Overview's Transactional card.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetTransactionalOverviewPeriod | Unset):  Default:
            GetTransactionalOverviewPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalOverview
     """


    return sync_detailed(
        workspace=workspace,
client=client,
period=period,
include_test=include_test,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: GetTransactionalOverviewPeriod | Unset = GetTransactionalOverviewPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Response[Problem | TransactionalOverview]:
    """ The period's numbers, the route, alerts, incidents and the keys that can send

     The Templates tab's header and side cards, and the Overview's Transactional card.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetTransactionalOverviewPeriod | Unset):  Default:
            GetTransactionalOverviewPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
include_test=include_test,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: GetTransactionalOverviewPeriod | Unset = GetTransactionalOverviewPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Problem | TransactionalOverview | None:
    """ The period's numbers, the route, alerts, incidents and the keys that can send

     The Templates tab's header and side cards, and the Overview's Transactional card.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (GetTransactionalOverviewPeriod | Unset):  Default:
            GetTransactionalOverviewPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalOverview
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
period=period,
include_test=include_test,

    )).parsed
