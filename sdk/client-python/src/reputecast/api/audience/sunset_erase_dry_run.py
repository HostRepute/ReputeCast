from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sunset_dry_run import SunsetDryRun
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    months: int | Unset = 12,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["months"] = months


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/audience/sunset-policy/dry-run".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SunsetDryRun:
    if response.status_code == 200:
        response_200 = SunsetDryRun.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SunsetDryRun]:
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
    months: int | Unset = 12,

) -> Response[Problem | SunsetDryRun]:
    """ How many contacts erasing after N months in Sunset would erase first (nothing changes)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        months (int | Unset):  Default: 12.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SunsetDryRun]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
months=months,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    months: int | Unset = 12,

) -> Problem | SunsetDryRun | None:
    """ How many contacts erasing after N months in Sunset would erase first (nothing changes)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        months (int | Unset):  Default: 12.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SunsetDryRun
     """


    return sync_detailed(
        workspace=workspace,
client=client,
months=months,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    months: int | Unset = 12,

) -> Response[Problem | SunsetDryRun]:
    """ How many contacts erasing after N months in Sunset would erase first (nothing changes)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        months (int | Unset):  Default: 12.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SunsetDryRun]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
months=months,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    months: int | Unset = 12,

) -> Problem | SunsetDryRun | None:
    """ How many contacts erasing after N months in Sunset would erase first (nothing changes)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        months (int | Unset):  Default: 12.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SunsetDryRun
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
months=months,

    )).parsed
