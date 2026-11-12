from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.traffic_rules import TrafficRules
from typing import cast



def _get_kwargs(
    workspace: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/traffic-rules".format(workspace=quote(str(workspace), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TrafficRules:
    if response.status_code == 200:
        response_200 = TrafficRules.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TrafficRules]:
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

) -> Response[Problem | TrafficRules]:
    """ A workspace's traffic rules: thresholds by bounce type, reply rules, canary, complaint watch

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TrafficRules]
     """


    kwargs = _get_kwargs(
        workspace=workspace,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> Problem | TrafficRules | None:
    """ A workspace's traffic rules: thresholds by bounce type, reply rules, canary, complaint watch

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TrafficRules
     """


    return sync_detailed(
        workspace=workspace,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | TrafficRules]:
    """ A workspace's traffic rules: thresholds by bounce type, reply rules, canary, complaint watch

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TrafficRules]
     """


    kwargs = _get_kwargs(
        workspace=workspace,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> Problem | TrafficRules | None:
    """ A workspace's traffic rules: thresholds by bounce type, reply rules, canary, complaint watch

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TrafficRules
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,

    )).parsed
