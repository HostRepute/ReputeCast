from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.website_campaign_web import WebsiteCampaignWeb
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/web".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebsiteCampaignWeb:
    if response.status_code == 200:
        response_200 = WebsiteCampaignWeb.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebsiteCampaignWeb]:
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

) -> Response[Problem | WebsiteCampaignWeb]:
    """ A campaign's Web tab: whether and how it goes on the website

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): The campaign's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteCampaignWeb]
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

) -> Problem | WebsiteCampaignWeb | None:
    """ A campaign's Web tab: whether and how it goes on the website

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): The campaign's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteCampaignWeb
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

) -> Response[Problem | WebsiteCampaignWeb]:
    """ A campaign's Web tab: whether and how it goes on the website

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): The campaign's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteCampaignWeb]
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

) -> Problem | WebsiteCampaignWeb | None:
    """ A campaign's Web tab: whether and how it goes on the website

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): The campaign's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteCampaignWeb
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,

    )).parsed
