from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.composer_version_list import ComposerVersionList
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    limit: int | Unset = 50,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/languages/{language}/versions".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),language=quote(str(language), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ComposerVersionList | Problem:
    if response.status_code == 200:
        response_200 = ComposerVersionList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ComposerVersionList | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 50,

) -> Response[ComposerVersionList | Problem]:
    """ Saved versions of a campaign's language version, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerVersionList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
language=language,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 50,

) -> ComposerVersionList | Problem | None:
    """ Saved versions of a campaign's language version, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerVersionList | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
language=language,
client=client,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 50,

) -> Response[ComposerVersionList | Problem]:
    """ Saved versions of a campaign's language version, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerVersionList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
language=language,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    limit: int | Unset = 50,

) -> ComposerVersionList | Problem | None:
    """ Saved versions of a campaign's language version, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerVersionList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
language=language,
client=client,
limit=limit,

    )).parsed
