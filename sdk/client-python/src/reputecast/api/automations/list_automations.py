from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_list import AutomationList
from ...models.list_automations_status import ListAutomationsStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    status: ListAutomationsStatus | Unset = ListAutomationsStatus.ALL,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/automations".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationList | Problem:
    if response.status_code == 200:
        response_200 = AutomationList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationList | Problem]:
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
    status: ListAutomationsStatus | Unset = ListAutomationsStatus.ALL,

) -> Response[AutomationList | Problem]:
    """ List automations with their 30-day performance

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListAutomationsStatus | Unset):  Default: ListAutomationsStatus.ALL.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListAutomationsStatus | Unset = ListAutomationsStatus.ALL,

) -> AutomationList | Problem | None:
    """ List automations with their 30-day performance

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListAutomationsStatus | Unset):  Default: ListAutomationsStatus.ALL.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListAutomationsStatus | Unset = ListAutomationsStatus.ALL,

) -> Response[AutomationList | Problem]:
    """ List automations with their 30-day performance

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListAutomationsStatus | Unset):  Default: ListAutomationsStatus.ALL.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListAutomationsStatus | Unset = ListAutomationsStatus.ALL,

) -> AutomationList | Problem | None:
    """ List automations with their 30-day performance

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListAutomationsStatus | Unset):  Default: ListAutomationsStatus.ALL.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,

    )).parsed
