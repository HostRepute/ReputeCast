from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.webhook_input import WebhookInput
from ...models.webhook_view import WebhookView
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    webhook_id: UUID,
    *,
    body: WebhookInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/webhooks/{webhook_id}".format(workspace=quote(str(workspace), safe=""),webhook_id=quote(str(webhook_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebhookView:
    if response.status_code == 200:
        response_200 = WebhookView.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebhookView]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WebhookInput,

) -> Response[Problem | WebhookView]:
    """ Change an endpoint's URL, name or events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):
        body (WebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
webhook_id=webhook_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WebhookInput,

) -> Problem | WebhookView | None:
    """ Change an endpoint's URL, name or events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):
        body (WebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookView
     """


    return sync_detailed(
        workspace=workspace,
webhook_id=webhook_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WebhookInput,

) -> Response[Problem | WebhookView]:
    """ Change an endpoint's URL, name or events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):
        body (WebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
webhook_id=webhook_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WebhookInput,

) -> Problem | WebhookView | None:
    """ Change an endpoint's URL, name or events

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):
        body (WebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookView
     """


    return (await asyncio_detailed(
        workspace=workspace,
webhook_id=webhook_id,
client=client,
body=body,

    )).parsed
