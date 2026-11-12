from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.mcp_agent_action_list import MCPAgentActionList
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    limit: int | Unset = 20,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/mcp-agents/actions".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> MCPAgentActionList | Problem:
    if response.status_code == 200:
        response_200 = MCPAgentActionList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[MCPAgentActionList | Problem]:
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
    limit: int | Unset = 20,

) -> Response[MCPAgentActionList | Problem]:
    """ Recent MCP agent tool calls from the audit log

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MCPAgentActionList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 20,

) -> MCPAgentActionList | Problem | None:
    """ Recent MCP agent tool calls from the audit log

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MCPAgentActionList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 20,

) -> Response[MCPAgentActionList | Problem]:
    """ Recent MCP agent tool calls from the audit log

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MCPAgentActionList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 20,

) -> MCPAgentActionList | Problem | None:
    """ Recent MCP agent tool calls from the audit log

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MCPAgentActionList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
limit=limit,

    )).parsed
