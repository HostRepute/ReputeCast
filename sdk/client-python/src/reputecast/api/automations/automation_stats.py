from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_stats import AutomationStats
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    id: str,
    *,
    days: int | Unset = 30,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["days"] = days


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/automations/{id}/stats".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationStats | Problem:
    if response.status_code == 200:
        response_200 = AutomationStats.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationStats | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    days: int | Unset = 30,

) -> Response[AutomationStats | Problem]:
    """ Live stats per step

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        days (int | Unset):  Default: 30.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationStats | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
days=days,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    days: int | Unset = 30,

) -> AutomationStats | Problem | None:
    """ Live stats per step

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        days (int | Unset):  Default: 30.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationStats | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
days=days,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    days: int | Unset = 30,

) -> Response[AutomationStats | Problem]:
    """ Live stats per step

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        days (int | Unset):  Default: 30.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationStats | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
days=days,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    days: int | Unset = 30,

) -> AutomationStats | Problem | None:
    """ Live stats per step

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        days (int | Unset):  Default: 30.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationStats | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
days=days,

    )).parsed
