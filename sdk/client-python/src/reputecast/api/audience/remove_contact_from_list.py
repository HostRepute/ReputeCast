from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact: UUID,
    list_: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/workspaces/{workspace}/contacts/{contact}/lists/{list_}".format(workspace=quote(str(workspace), safe=""),contact=quote(str(contact), safe=""),list_=quote(str(list_), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Problem:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    contact: UUID,
    list_: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Any | Problem]:
    """ Take a contact off a list (the team's change, not an unsubscribe)

     Recorded in the contact's history as removed by the team; audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        list_ (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
list_=list_,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    contact: UUID,
    list_: UUID,
    *,
    client: AuthenticatedClient,

) -> Any | Problem | None:
    """ Take a contact off a list (the team's change, not an unsubscribe)

     Recorded in the contact's history as removed by the team; audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        list_ (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return sync_detailed(
        workspace=workspace,
contact=contact,
list_=list_,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact: UUID,
    list_: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Any | Problem]:
    """ Take a contact off a list (the team's change, not an unsubscribe)

     Recorded in the contact's history as removed by the team; audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        list_ (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
list_=list_,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    contact: UUID,
    list_: UUID,
    *,
    client: AuthenticatedClient,

) -> Any | Problem | None:
    """ Take a contact off a list (the team's change, not an unsubscribe)

     Recorded in the contact's history as removed by the team; audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        list_ (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact=contact,
list_=list_,
client=client,

    )).parsed
