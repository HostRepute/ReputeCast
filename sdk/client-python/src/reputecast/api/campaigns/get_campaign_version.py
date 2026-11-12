from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.composer_version_detail import ComposerVersionDetail
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    version: int,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/versions/{version}".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),version=quote(str(version), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ComposerVersionDetail | Problem:
    if response.status_code == 200:
        response_200 = ComposerVersionDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ComposerVersionDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> Response[ComposerVersionDetail | Problem]:
    """ One version with its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerVersionDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
version=version,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> ComposerVersionDetail | Problem | None:
    """ One version with its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerVersionDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
version=version,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> Response[ComposerVersionDetail | Problem]:
    """ One version with its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerVersionDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
version=version,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    client: AuthenticatedClient,

) -> ComposerVersionDetail | Problem | None:
    """ One version with its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerVersionDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
version=version,
client=client,

    )).parsed
