from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.composer_readiness import ComposerReadiness
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/readiness/composer".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ComposerReadiness | Problem:
    if response.status_code == 200:
        response_200 = ComposerReadiness.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ComposerReadiness | Problem]:
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

) -> Response[ComposerReadiness | Problem]:
    """ Workspace rules and sender reputation for the composer's checks

     From the campaign's latest Send Readiness run, without running the checks: how the workspace's
    sending rules and the sender's reputation stand, their warnings and blocks, and when the blocklists
    were checked.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerReadiness | Problem]
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

) -> ComposerReadiness | Problem | None:
    """ Workspace rules and sender reputation for the composer's checks

     From the campaign's latest Send Readiness run, without running the checks: how the workspace's
    sending rules and the sender's reputation stand, their warnings and blocks, and when the blocklists
    were checked.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerReadiness | Problem
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

) -> Response[ComposerReadiness | Problem]:
    """ Workspace rules and sender reputation for the composer's checks

     From the campaign's latest Send Readiness run, without running the checks: how the workspace's
    sending rules and the sender's reputation stand, their warnings and blocks, and when the blocklists
    were checked.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerReadiness | Problem]
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

) -> ComposerReadiness | Problem | None:
    """ Workspace rules and sender reputation for the composer's checks

     From the campaign's latest Send Readiness run, without running the checks: how the workspace's
    sending rules and the sender's reputation stand, their warnings and blocks, and when the blocklists
    were checked.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerReadiness | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,

    )).parsed
