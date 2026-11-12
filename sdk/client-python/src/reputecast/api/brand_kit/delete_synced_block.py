from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.synced_deleted import SyncedDeleted
from typing import cast



def _get_kwargs(
    workspace: str,
    kind: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/workspaces/{workspace}/brand-kit/synced/{kind}".format(workspace=quote(str(workspace), safe=""),kind=quote(str(kind), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SyncedDeleted:
    if response.status_code == 200:
        response_200 = SyncedDeleted.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SyncedDeleted]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    kind: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SyncedDeleted]:
    """ Delete a named synced block

     Drafts and scheduled emails holding it keep its content as ordinary sections, in a new version;
    emails in an approval, sending, paused or sent keep theirs. The Header and Footer can't be deleted.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): A named block's key (the Header and Footer can't be deleted)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SyncedDeleted]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    kind: str,
    *,
    client: AuthenticatedClient,

) -> Problem | SyncedDeleted | None:
    """ Delete a named synced block

     Drafts and scheduled emails holding it keep its content as ordinary sections, in a new version;
    emails in an approval, sending, paused or sent keep theirs. The Header and Footer can't be deleted.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): A named block's key (the Header and Footer can't be deleted)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SyncedDeleted
     """


    return sync_detailed(
        workspace=workspace,
kind=kind,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    kind: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SyncedDeleted]:
    """ Delete a named synced block

     Drafts and scheduled emails holding it keep its content as ordinary sections, in a new version;
    emails in an approval, sending, paused or sent keep theirs. The Header and Footer can't be deleted.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): A named block's key (the Header and Footer can't be deleted)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SyncedDeleted]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    kind: str,
    *,
    client: AuthenticatedClient,

) -> Problem | SyncedDeleted | None:
    """ Delete a named synced block

     Drafts and scheduled emails holding it keep its content as ordinary sections, in a new version;
    emails in an approval, sending, paused or sent keep theirs. The Header and Footer can't be deleted.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): A named block's key (the Header and Footer can't be deleted)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SyncedDeleted
     """


    return (await asyncio_detailed(
        workspace=workspace,
kind=kind,
client=client,

    )).parsed
