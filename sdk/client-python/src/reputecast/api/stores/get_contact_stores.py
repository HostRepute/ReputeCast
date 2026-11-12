from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.commerce_contact_stores import CommerceContactStores
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/contacts/{contact_id}/stores".format(workspace=quote(str(workspace), safe=""),contact_id=quote(str(contact_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CommerceContactStores | Problem:
    if response.status_code == 200:
        response_200 = CommerceContactStores.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CommerceContactStores | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[CommerceContactStores | Problem]:
    """ The contact's store orders and their consent in each store

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID): The contact's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CommerceContactStores | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact_id=contact_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,

) -> CommerceContactStores | Problem | None:
    """ The contact's store orders and their consent in each store

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID): The contact's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CommerceContactStores | Problem
     """


    return sync_detailed(
        workspace=workspace,
contact_id=contact_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[CommerceContactStores | Problem]:
    """ The contact's store orders and their consent in each store

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID): The contact's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CommerceContactStores | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact_id=contact_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,

) -> CommerceContactStores | Problem | None:
    """ The contact's store orders and their consent in each store

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID): The contact's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CommerceContactStores | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact_id=contact_id,
client=client,

    )).parsed
