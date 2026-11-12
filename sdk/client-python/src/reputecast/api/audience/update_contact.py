from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.contact_profile import ContactProfile
from ...models.contact_update import ContactUpdate
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact: UUID,
    *,
    body: ContactUpdate,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/workspaces/{workspace}/contacts/{contact}".format(workspace=quote(str(workspace), safe=""),contact=quote(str(contact), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ContactProfile | Problem:
    if response.status_code == 200:
        response_200 = ContactProfile.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ContactProfile | Problem]:
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
    body: ContactUpdate,

) -> Response[ContactProfile | Problem]:
    """ Change a contact's fields and tags

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (ContactUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactProfile | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
body=body,

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
    body: ContactUpdate,

) -> ContactProfile | Problem | None:
    """ Change a contact's fields and tags

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (ContactUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactProfile | Problem
     """


    return sync_detailed(
        workspace=workspace,
contact=contact,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,
    body: ContactUpdate,

) -> Response[ContactProfile | Problem]:
    """ Change a contact's fields and tags

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (ContactUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactProfile | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
body=body,

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
    body: ContactUpdate,

) -> ContactProfile | Problem | None:
    """ Change a contact's fields and tags

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (ContactUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactProfile | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact=contact,
client=client,
body=body,

    )).parsed
