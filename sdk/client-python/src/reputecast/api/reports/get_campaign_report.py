from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.campaign_report import CampaignReport
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    language: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["language"] = language


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/report".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CampaignReport | Problem:
    if response.status_code == 200:
        response_200 = CampaignReport.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CampaignReport | Problem]:
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
    language: str | Unset = UNSET,

) -> Response[CampaignReport | Problem]:
    """ Campaign report: headline metrics, 72-hour engagement, click map, providers, bounces, unsubscribes,
    A/B

     Aggregate numbers only. Rates are percentages: delivered and bounces of sent; opens, clicks,
    unsubscribes and complaints of delivered.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        language (str | Unset): Show the click map of one language version (campaigns sent in
            several languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignReport | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
language=language,

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
    language: str | Unset = UNSET,

) -> CampaignReport | Problem | None:
    """ Campaign report: headline metrics, 72-hour engagement, click map, providers, bounces, unsubscribes,
    A/B

     Aggregate numbers only. Rates are percentages: delivered and bounces of sent; opens, clicks,
    unsubscribes and complaints of delivered.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        language (str | Unset): Show the click map of one language version (campaigns sent in
            several languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignReport | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
language=language,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> Response[CampaignReport | Problem]:
    """ Campaign report: headline metrics, 72-hour engagement, click map, providers, bounces, unsubscribes,
    A/B

     Aggregate numbers only. Rates are percentages: delivered and bounces of sent; opens, clicks,
    unsubscribes and complaints of delivered.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        language (str | Unset): Show the click map of one language version (campaigns sent in
            several languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignReport | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
language=language,

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
    language: str | Unset = UNSET,

) -> CampaignReport | Problem | None:
    """ Campaign report: headline metrics, 72-hour engagement, click map, providers, bounces, unsubscribes,
    A/B

     Aggregate numbers only. Rates are percentages: delivered and bounces of sent; opens, clicks,
    unsubscribes and complaints of delivered.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        language (str | Unset): Show the click map of one language version (campaigns sent in
            several languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignReport | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
language=language,

    )).parsed
