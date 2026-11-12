from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.rendering_connect_input import RenderingConnectInput
from ...models.rendering_service import RenderingService
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: RenderingConnectInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/rendering-service".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | RenderingService:
    if response.status_code == 200:
        response_200 = RenderingService.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | RenderingService]:
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
    body: RenderingConnectInput,

) -> Response[Problem | RenderingService]:
    """ Connect Litmus or Mailgun Inspect with the workspace's own API key

     The key is tested by listing the account's email clients, then sealed at rest. It is never returned;
    a new key replaces the old one.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RenderingConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RenderingService]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: RenderingConnectInput,

) -> Problem | RenderingService | None:
    """ Connect Litmus or Mailgun Inspect with the workspace's own API key

     The key is tested by listing the account's email clients, then sealed at rest. It is never returned;
    a new key replaces the old one.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RenderingConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RenderingService
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: RenderingConnectInput,

) -> Response[Problem | RenderingService]:
    """ Connect Litmus or Mailgun Inspect with the workspace's own API key

     The key is tested by listing the account's email clients, then sealed at rest. It is never returned;
    a new key replaces the old one.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RenderingConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RenderingService]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: RenderingConnectInput,

) -> Problem | RenderingService | None:
    """ Connect Litmus or Mailgun Inspect with the workspace's own API key

     The key is tested by listing the account's email clients, then sealed at rest. It is never returned;
    a new key replaces the old one.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RenderingConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RenderingService
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
