from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.tracker_recent_events import TrackerRecentEvents
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    site: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/tracker/sites/{site}/events/recent".format(workspace=quote(str(workspace), safe=""),site=quote(str(site), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TrackerRecentEvents:
    if response.status_code == 200:
        response_200 = TrackerRecentEvents.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TrackerRecentEvents]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    site: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | TrackerRecentEvents]:
    """ A tracked site's last 20 events

     What happened, on which page or product, whether a known contact or a visitor, and when. Never an
    address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        site (UUID): Tracked site id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TrackerRecentEvents]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
site=site,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    site: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | TrackerRecentEvents | None:
    """ A tracked site's last 20 events

     What happened, on which page or product, whether a known contact or a visitor, and when. Never an
    address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        site (UUID): Tracked site id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TrackerRecentEvents
     """


    return sync_detailed(
        workspace=workspace,
site=site,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    site: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | TrackerRecentEvents]:
    """ A tracked site's last 20 events

     What happened, on which page or product, whether a known contact or a visitor, and when. Never an
    address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        site (UUID): Tracked site id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TrackerRecentEvents]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
site=site,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    site: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | TrackerRecentEvents | None:
    """ A tracked site's last 20 events

     What happened, on which page or product, whether a known contact or a visitor, and when. Never an
    address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        site (UUID): Tracked site id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TrackerRecentEvents
     """


    return (await asyncio_detailed(
        workspace=workspace,
site=site,
client=client,

    )).parsed
