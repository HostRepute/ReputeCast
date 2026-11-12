from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_holdout_results_window import GetHoldoutResultsWindow
from ...models.holdout_results import HoldoutResults
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    *,
    period: UUID | Unset = UNSET,
    window: GetHoldoutResultsWindow | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = str(period)
    params["period"] = json_period

    json_window: str | Unset = UNSET
    if not isinstance(window, Unset):
        json_window = window.value

    params["window"] = json_window


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/holdout/results".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> HoldoutResults | Problem:
    if response.status_code == 200:
        response_200 = HoldoutResults.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[HoldoutResults | Problem]:
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
    period: UUID | Unset = UNSET,
    window: GetHoldoutResultsWindow | Unset = UNSET,

) -> Response[HoldoutResults | Problem]:
    """ What marketing added, measured against the holdout group

     The holdout group's and everyone else's conversion rate, revenue and orders per contact, with the
    lift and its confidence. Measured every night at 04:00 workspace time.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (UUID | Unset): A period's id; the newest by default
        window (GetHoldoutResultsWindow | Unset): The whole period (default) or its last 30 or 90
            days

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[HoldoutResults | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
window=window,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: UUID | Unset = UNSET,
    window: GetHoldoutResultsWindow | Unset = UNSET,

) -> HoldoutResults | Problem | None:
    """ What marketing added, measured against the holdout group

     The holdout group's and everyone else's conversion rate, revenue and orders per contact, with the
    lift and its confidence. Measured every night at 04:00 workspace time.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (UUID | Unset): A period's id; the newest by default
        window (GetHoldoutResultsWindow | Unset): The whole period (default) or its last 30 or 90
            days

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        HoldoutResults | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
period=period,
window=window,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: UUID | Unset = UNSET,
    window: GetHoldoutResultsWindow | Unset = UNSET,

) -> Response[HoldoutResults | Problem]:
    """ What marketing added, measured against the holdout group

     The holdout group's and everyone else's conversion rate, revenue and orders per contact, with the
    lift and its confidence. Measured every night at 04:00 workspace time.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (UUID | Unset): A period's id; the newest by default
        window (GetHoldoutResultsWindow | Unset): The whole period (default) or its last 30 or 90
            days

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[HoldoutResults | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
window=window,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: UUID | Unset = UNSET,
    window: GetHoldoutResultsWindow | Unset = UNSET,

) -> HoldoutResults | Problem | None:
    """ What marketing added, measured against the holdout group

     The holdout group's and everyone else's conversion rate, revenue and orders per contact, with the
    lift and its confidence. Measured every night at 04:00 workspace time.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (UUID | Unset): A period's id; the newest by default
        window (GetHoldoutResultsWindow | Unset): The whole period (default) or its last 30 or 90
            days

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        HoldoutResults | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
period=period,
window=window,

    )).parsed
