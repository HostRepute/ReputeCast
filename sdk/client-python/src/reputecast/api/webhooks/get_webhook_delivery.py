from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.webhook_delivery_detail import WebhookDeliveryDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    delivery_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/webhooks/deliveries/{delivery_id}".format(workspace=quote(str(workspace), safe=""),delivery_id=quote(str(delivery_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebhookDeliveryDetail:
    if response.status_code == 200:
        response_200 = WebhookDeliveryDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebhookDeliveryDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    delivery_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | WebhookDeliveryDetail]:
    """ One delivery with its payload and the endpoint's response

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        delivery_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookDeliveryDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
delivery_id=delivery_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    delivery_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | WebhookDeliveryDetail | None:
    """ One delivery with its payload and the endpoint's response

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        delivery_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookDeliveryDetail
     """


    return sync_detailed(
        workspace=workspace,
delivery_id=delivery_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    delivery_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | WebhookDeliveryDetail]:
    """ One delivery with its payload and the endpoint's response

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        delivery_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebhookDeliveryDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
delivery_id=delivery_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    delivery_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | WebhookDeliveryDetail | None:
    """ One delivery with its payload and the endpoint's response

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        delivery_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebhookDeliveryDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
delivery_id=delivery_id,
client=client,

    )).parsed
