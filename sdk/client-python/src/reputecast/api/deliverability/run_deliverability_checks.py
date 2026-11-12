from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.deliverability_check_run import DeliverabilityCheckRun
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    builtin: bool | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["builtin"] = builtin


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/deliverability/checks".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DeliverabilityCheckRun | Problem:
    if response.status_code == 202:
        response_202 = DeliverabilityCheckRun.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DeliverabilityCheckRun | Problem]:
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
    builtin: bool | Unset = UNSET,

) -> Response[DeliverabilityCheckRun | Problem]:
    """ Run all checks now

     Asks HostRepute to re-check every source when it is connected, otherwise runs the built-in checks.
    Returns the run in progress when one is already running.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        builtin (bool | Unset): Run the built-in checks even when HostRepute is connected (while
            HostRepute is unreachable)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityCheckRun | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
builtin=builtin,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    builtin: bool | Unset = UNSET,

) -> DeliverabilityCheckRun | Problem | None:
    """ Run all checks now

     Asks HostRepute to re-check every source when it is connected, otherwise runs the built-in checks.
    Returns the run in progress when one is already running.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        builtin (bool | Unset): Run the built-in checks even when HostRepute is connected (while
            HostRepute is unreachable)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityCheckRun | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
builtin=builtin,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    builtin: bool | Unset = UNSET,

) -> Response[DeliverabilityCheckRun | Problem]:
    """ Run all checks now

     Asks HostRepute to re-check every source when it is connected, otherwise runs the built-in checks.
    Returns the run in progress when one is already running.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        builtin (bool | Unset): Run the built-in checks even when HostRepute is connected (while
            HostRepute is unreachable)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityCheckRun | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
builtin=builtin,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    builtin: bool | Unset = UNSET,

) -> DeliverabilityCheckRun | Problem | None:
    """ Run all checks now

     Asks HostRepute to re-check every source when it is connected, otherwise runs the built-in checks.
    Returns the run in progress when one is already running.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        builtin (bool | Unset): Run the built-in checks even when HostRepute is connected (while
            HostRepute is unreachable)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityCheckRun | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
builtin=builtin,

    )).parsed
