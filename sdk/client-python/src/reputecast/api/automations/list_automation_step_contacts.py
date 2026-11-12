from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_step_contacts import AutomationStepContacts
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    id: str,
    node: str,
    *,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/automations/{id}/steps/{node}/contacts".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),node=quote(str(node), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationStepContacts | Problem:
    if response.status_code == 200:
        response_200 = AutomationStepContacts.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationStepContacts | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: str,
    node: str,
    *,
    client: AuthenticatedClient,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> Response[AutomationStepContacts | Problem]:
    """ The contacts at a step, the latest to arrive first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        node (str):
        cursor (str | Unset):
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationStepContacts | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
node=node,
cursor=cursor,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: str,
    node: str,
    *,
    client: AuthenticatedClient,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> AutomationStepContacts | Problem | None:
    """ The contacts at a step, the latest to arrive first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        node (str):
        cursor (str | Unset):
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationStepContacts | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
node=node,
client=client,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: str,
    node: str,
    *,
    client: AuthenticatedClient,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> Response[AutomationStepContacts | Problem]:
    """ The contacts at a step, the latest to arrive first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        node (str):
        cursor (str | Unset):
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationStepContacts | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
node=node,
cursor=cursor,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: str,
    node: str,
    *,
    client: AuthenticatedClient,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> AutomationStepContacts | Problem | None:
    """ The contacts at a step, the latest to arrive first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        node (str):
        cursor (str | Unset):
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationStepContacts | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
node=node,
client=client,
cursor=cursor,
limit=limit,

    )).parsed
