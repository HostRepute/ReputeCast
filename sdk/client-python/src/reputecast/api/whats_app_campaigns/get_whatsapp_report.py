from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.whats_app_report import WhatsAppReport
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaign_id}/report".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WhatsAppReport:
    if response.status_code == 200:
        response_200 = WhatsAppReport.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WhatsAppReport]:
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

) -> Response[Problem | WhatsAppReport]:
    """ The WhatsApp report: sent, delivered, read, failed, button clicks, replies, opt-outs and cost

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppReport]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,

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

) -> Problem | WhatsAppReport | None:
    """ The WhatsApp report: sent, delivered, read, failed, button clicks, replies, opt-outs and cost

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppReport
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | WhatsAppReport]:
    """ The WhatsApp report: sent, delivered, read, failed, button clicks, replies, opt-outs and cost

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppReport]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,

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

) -> Problem | WhatsAppReport | None:
    """ The WhatsApp report: sent, delivered, read, failed, button clicks, replies, opt-outs and cost

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppReport
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,

    )).parsed
