from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.whats_app_campaign import WhatsAppCampaign
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    preview: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["preview"] = preview


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaign_id}".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WhatsAppCampaign:
    if response.status_code == 200:
        response_200 = WhatsAppCampaign.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WhatsAppCampaign]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    preview: str | Unset = UNSET,

) -> Response[Problem | WhatsAppCampaign]:
    """ The WhatsApp composer

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        preview (str | Unset): The contact the preview shows

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppCampaign]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
preview=preview,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    preview: str | Unset = UNSET,

) -> Problem | WhatsAppCampaign | None:
    """ The WhatsApp composer

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        preview (str | Unset): The contact the preview shows

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppCampaign
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
preview=preview,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    preview: str | Unset = UNSET,

) -> Response[Problem | WhatsAppCampaign]:
    """ The WhatsApp composer

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        preview (str | Unset): The contact the preview shows

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppCampaign]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
preview=preview,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    preview: str | Unset = UNSET,

) -> Problem | WhatsAppCampaign | None:
    """ The WhatsApp composer

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        preview (str | Unset): The contact the preview shows

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppCampaign
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
preview=preview,

    )).parsed
