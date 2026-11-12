from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.website_campaign_states import WebsiteCampaignStates
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    ids: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["ids"] = ids


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns-web".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebsiteCampaignStates:
    if response.status_code == 200:
        response_200 = WebsiteCampaignStates.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebsiteCampaignStates]:
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
    ids: str | Unset = UNSET,

) -> Response[Problem | WebsiteCampaignStates]:
    """ The Campaigns list's Web column: the posts of campaigns

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        ids (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteCampaignStates]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
ids=ids,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    ids: str | Unset = UNSET,

) -> Problem | WebsiteCampaignStates | None:
    """ The Campaigns list's Web column: the posts of campaigns

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        ids (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteCampaignStates
     """


    return sync_detailed(
        workspace=workspace,
client=client,
ids=ids,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    ids: str | Unset = UNSET,

) -> Response[Problem | WebsiteCampaignStates]:
    """ The Campaigns list's Web column: the posts of campaigns

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        ids (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteCampaignStates]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
ids=ids,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    ids: str | Unset = UNSET,

) -> Problem | WebsiteCampaignStates | None:
    """ The Campaigns list's Web column: the posts of campaigns

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        ids (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteCampaignStates
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
ids=ids,

    )).parsed
