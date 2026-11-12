from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.preference_link_result import PreferenceLinkResult
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/contacts/{contact}/preference-link".format(workspace=quote(str(workspace), safe=""),contact=quote(str(contact), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PreferenceLinkResult | Problem:
    if response.status_code == 200:
        response_200 = PreferenceLinkResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PreferenceLinkResult | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PreferenceLinkResult | Problem]:
    """ Get the contact's preference center link (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PreferenceLinkResult | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> PreferenceLinkResult | Problem | None:
    """ Get the contact's preference center link (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PreferenceLinkResult | Problem
     """


    return sync_detailed(
        workspace=workspace,
contact=contact,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PreferenceLinkResult | Problem]:
    """ Get the contact's preference center link (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PreferenceLinkResult | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> PreferenceLinkResult | Problem | None:
    """ Get the contact's preference center link (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PreferenceLinkResult | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact=contact,
client=client,

    )).parsed
