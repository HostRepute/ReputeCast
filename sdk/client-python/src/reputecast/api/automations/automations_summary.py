from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_summary import AutomationSummary
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    days: int | Unset = 7,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["days"] = days


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/automations/summary".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationSummary | Problem:
    if response.status_code == 200:
        response_200 = AutomationSummary.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationSummary | Problem]:
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
    days: int | Unset = 7,

) -> Response[AutomationSummary | Problem]:
    """ Contacts in flows now and the last N days of automation email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        days (int | Unset):  Default: 7.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationSummary | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
days=days,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    days: int | Unset = 7,

) -> AutomationSummary | Problem | None:
    """ Contacts in flows now and the last N days of automation email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        days (int | Unset):  Default: 7.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationSummary | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
days=days,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    days: int | Unset = 7,

) -> Response[AutomationSummary | Problem]:
    """ Contacts in flows now and the last N days of automation email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        days (int | Unset):  Default: 7.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationSummary | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
days=days,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    days: int | Unset = 7,

) -> AutomationSummary | Problem | None:
    """ Contacts in flows now and the last N days of automation email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        days (int | Unset):  Default: 7.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationSummary | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
days=days,

    )).parsed
