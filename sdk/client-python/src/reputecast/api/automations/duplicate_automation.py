from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_created import AutomationCreated
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    id: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/automations/{id}/duplicate".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationCreated | Problem:
    if response.status_code == 201:
        response_201 = AutomationCreated.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationCreated | Problem]:
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

) -> Response[AutomationCreated | Problem]:
    """ Copy an automation's draft into a new draft

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationCreated | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,

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

) -> AutomationCreated | Problem | None:
    """ Copy an automation's draft into a new draft

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationCreated | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,

) -> Response[AutomationCreated | Problem]:
    """ Copy an automation's draft into a new draft

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationCreated | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,

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

) -> AutomationCreated | Problem | None:
    """ Copy an automation's draft into a new draft

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationCreated | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,

    )).parsed
