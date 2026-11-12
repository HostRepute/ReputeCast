from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.connect_setup import ConnectSetup
from ...models.connect_webhook_input import ConnectWebhookInput
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    setup_id: UUID,
    *,
    body: ConnectWebhookInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/connect/setups/{setup_id}/webhook".format(workspace=quote(str(workspace), safe=""),setup_id=quote(str(setup_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ConnectSetup | Problem:
    if response.status_code == 200:
        response_200 = ConnectSetup.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ConnectSetup | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ConnectWebhookInput,

) -> Response[ConnectSetup | Problem]:
    """ Test the webhook endpoint again (optionally through a tunnel URL) and register it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):
        body (ConnectWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectSetup | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
setup_id=setup_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ConnectWebhookInput,

) -> ConnectSetup | Problem | None:
    """ Test the webhook endpoint again (optionally through a tunnel URL) and register it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):
        body (ConnectWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectSetup | Problem
     """


    return sync_detailed(
        workspace=workspace,
setup_id=setup_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ConnectWebhookInput,

) -> Response[ConnectSetup | Problem]:
    """ Test the webhook endpoint again (optionally through a tunnel URL) and register it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):
        body (ConnectWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectSetup | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
setup_id=setup_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ConnectWebhookInput,

) -> ConnectSetup | Problem | None:
    """ Test the webhook endpoint again (optionally through a tunnel URL) and register it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):
        body (ConnectWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectSetup | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
setup_id=setup_id,
client=client,
body=body,

    )).parsed
