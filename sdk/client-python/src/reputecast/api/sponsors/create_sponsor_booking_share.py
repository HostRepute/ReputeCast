from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sponsor_share_link import SponsorShareLink
from ...models.sponsors_share_input import SponsorsShareInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    body: SponsorsShareInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/sponsor-bookings/{id}/shares".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SponsorShareLink:
    if response.status_code == 201:
        response_201 = SponsorShareLink.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SponsorShareLink]:
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
    body: SponsorsShareInput,

) -> Response[Problem | SponsorShareLink]:
    """ Share a booking's report as a read-only link, optionally expiring

     The page shows only the sponsor's numbers and the creative as sent, never contact data.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): Id
        body (SponsorsShareInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorShareLink]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,

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
    body: SponsorsShareInput,

) -> Problem | SponsorShareLink | None:
    """ Share a booking's report as a read-only link, optionally expiring

     The page shows only the sponsor's numbers and the creative as sent, never contact data.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): Id
        body (SponsorsShareInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorShareLink
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: SponsorsShareInput,

) -> Response[Problem | SponsorShareLink]:
    """ Share a booking's report as a read-only link, optionally expiring

     The page shows only the sponsor's numbers and the creative as sent, never contact data.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): Id
        body (SponsorsShareInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorShareLink]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,

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
    body: SponsorsShareInput,

) -> Problem | SponsorShareLink | None:
    """ Share a booking's report as a read-only link, optionally expiring

     The page shows only the sponsor's numbers and the creative as sent, never contact data.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): Id
        body (SponsorsShareInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorShareLink
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,

    )).parsed
