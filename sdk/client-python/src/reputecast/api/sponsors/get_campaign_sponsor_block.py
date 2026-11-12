from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_campaign_sponsor_block_label import GetCampaignSponsorBlockLabel
from ...models.get_campaign_sponsor_block_layout import GetCampaignSponsorBlockLayout
from ...models.problem import Problem
from ...models.sponsor_block import SponsorBlock
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    layout: GetCampaignSponsorBlockLayout | Unset = UNSET,
    label: GetCampaignSponsorBlockLabel | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_layout: str | Unset = UNSET
    if not isinstance(layout, Unset):
        json_layout = layout.value

    params["layout"] = json_layout

    json_label: str | Unset = UNSET
    if not isinstance(label, Unset):
        json_label = label.value

    params["label"] = json_label


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{id}/sponsor-block".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SponsorBlock:
    if response.status_code == 200:
        response_200 = SponsorBlock.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SponsorBlock]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    layout: GetCampaignSponsorBlockLayout | Unset = UNSET,
    label: GetCampaignSponsorBlockLabel | Unset = UNSET,

) -> Response[Problem | SponsorBlock]:
    """ The Sponsor block of a campaign: its booking, creative and content

     The price shows only to people who hold earn:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The campaign's id
        layout (GetCampaignSponsorBlockLayout | Unset): The block's layout
        label (GetCampaignSponsorBlockLabel | Unset): Plain "Sponsored", or "Sponsored by" the
            sponsor

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorBlock]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
layout=layout,
label=label,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    layout: GetCampaignSponsorBlockLayout | Unset = UNSET,
    label: GetCampaignSponsorBlockLabel | Unset = UNSET,

) -> Problem | SponsorBlock | None:
    """ The Sponsor block of a campaign: its booking, creative and content

     The price shows only to people who hold earn:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The campaign's id
        layout (GetCampaignSponsorBlockLayout | Unset): The block's layout
        label (GetCampaignSponsorBlockLabel | Unset): Plain "Sponsored", or "Sponsored by" the
            sponsor

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorBlock
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
layout=layout,
label=label,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    layout: GetCampaignSponsorBlockLayout | Unset = UNSET,
    label: GetCampaignSponsorBlockLabel | Unset = UNSET,

) -> Response[Problem | SponsorBlock]:
    """ The Sponsor block of a campaign: its booking, creative and content

     The price shows only to people who hold earn:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The campaign's id
        layout (GetCampaignSponsorBlockLayout | Unset): The block's layout
        label (GetCampaignSponsorBlockLabel | Unset): Plain "Sponsored", or "Sponsored by" the
            sponsor

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorBlock]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
layout=layout,
label=label,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    layout: GetCampaignSponsorBlockLayout | Unset = UNSET,
    label: GetCampaignSponsorBlockLabel | Unset = UNSET,

) -> Problem | SponsorBlock | None:
    """ The Sponsor block of a campaign: its booking, creative and content

     The price shows only to people who hold earn:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The campaign's id
        layout (GetCampaignSponsorBlockLayout | Unset): The block's layout
        label (GetCampaignSponsorBlockLabel | Unset): Plain "Sponsored", or "Sponsored by" the
            sponsor

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorBlock
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
layout=layout,
label=label,

    )).parsed
