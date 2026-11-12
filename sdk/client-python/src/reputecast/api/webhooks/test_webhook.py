from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.webhook_test_result import WebhookTestResult
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    webhook_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/webhooks/{webhook_id}/test".format(workspace=quote(str(workspace), safe=""),webhook_id=quote(str(webhook_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebhookTestResult:
    if response.status_code == 200:
        response_200 = WebhookTestResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebhookTestResult]:
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

) -> Response[Problem | WebhookTestResult]:
    """ Send a webhook.test event now and return the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookTestResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
webhook_id=webhook_id,

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

) -> Problem | WebhookTestResult | None:
    """ Send a webhook.test event now and return the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookTestResult
     """


    return sync_detailed(
        workspace=workspace,
webhook_id=webhook_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | WebhookTestResult]:
    """ Send a webhook.test event now and return the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookTestResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
webhook_id=webhook_id,

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

) -> Problem | WebhookTestResult | None:
    """ Send a webhook.test event now and return the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookTestResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
webhook_id=webhook_id,
client=client,

    )).parsed
