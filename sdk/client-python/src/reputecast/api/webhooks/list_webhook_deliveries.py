from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_webhook_deliveries_state import ListWebhookDeliveriesState
from ...models.problem import Problem
from ...models.webhook_delivery_list import WebhookDeliveryList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    webhook_id: str | Unset = UNSET,
    state: ListWebhookDeliveriesState | Unset = UNSET,
    limit: int | Unset = 50,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["webhookId"] = webhook_id

    json_state: str | Unset = UNSET
    if not isinstance(state, Unset):
        json_state = state.value

    params["state"] = json_state

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/webhooks/deliveries".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebhookDeliveryList:
    if response.status_code == 200:
        response_200 = WebhookDeliveryList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebhookDeliveryList]:
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
    webhook_id: str | Unset = UNSET,
    state: ListWebhookDeliveriesState | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[Problem | WebhookDeliveryList]:
    """ The delivery log, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (str | Unset): Only this endpoint
        state (ListWebhookDeliveriesState | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookDeliveryList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
webhook_id=webhook_id,
state=state,
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
    webhook_id: str | Unset = UNSET,
    state: ListWebhookDeliveriesState | Unset = UNSET,
    limit: int | Unset = 50,

) -> Problem | WebhookDeliveryList | None:
    """ The delivery log, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (str | Unset): Only this endpoint
        state (ListWebhookDeliveriesState | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookDeliveryList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
webhook_id=webhook_id,
state=state,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    webhook_id: str | Unset = UNSET,
    state: ListWebhookDeliveriesState | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[Problem | WebhookDeliveryList]:
    """ The delivery log, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (str | Unset): Only this endpoint
        state (ListWebhookDeliveriesState | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookDeliveryList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
webhook_id=webhook_id,
state=state,
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
    webhook_id: str | Unset = UNSET,
    state: ListWebhookDeliveriesState | Unset = UNSET,
    limit: int | Unset = 50,

) -> Problem | WebhookDeliveryList | None:
    """ The delivery log, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        webhook_id (str | Unset): Only this endpoint
        state (ListWebhookDeliveriesState | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookDeliveryList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
webhook_id=webhook_id,
state=state,
limit=limit,

    )).parsed
