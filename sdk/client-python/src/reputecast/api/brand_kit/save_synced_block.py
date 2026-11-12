from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.synced_input import SyncedInput
from ...models.synced_saved import SyncedSaved
from typing import cast



def _get_kwargs(
    workspace: str,
    kind: str,
    *,
    body: SyncedInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/brand-kit/synced/{kind}".format(workspace=quote(str(workspace), safe=""),kind=quote(str(kind), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SyncedSaved:
    if response.status_code == 200:
        response_200 = SyncedSaved.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SyncedSaved]:
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
    body: SyncedInput,

) -> Response[Problem | SyncedSaved]:
    """ Change a synced block (the Header, the Footer or a named block)

     Drafts and scheduled emails holding the block get the change as a new version; emails in an
    approval, sending, paused or sent keep theirs. A named block can be renamed with it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): header, footer, or a named block's key
        body (SyncedInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SyncedSaved]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,
body=body,

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
    body: SyncedInput,

) -> Problem | SyncedSaved | None:
    """ Change a synced block (the Header, the Footer or a named block)

     Drafts and scheduled emails holding the block get the change as a new version; emails in an
    approval, sending, paused or sent keep theirs. A named block can be renamed with it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): header, footer, or a named block's key
        body (SyncedInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SyncedSaved
     """


    return sync_detailed(
        workspace=workspace,
kind=kind,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    kind: str,
    *,
    client: AuthenticatedClient,
    body: SyncedInput,

) -> Response[Problem | SyncedSaved]:
    """ Change a synced block (the Header, the Footer or a named block)

     Drafts and scheduled emails holding the block get the change as a new version; emails in an
    approval, sending, paused or sent keep theirs. A named block can be renamed with it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): header, footer, or a named block's key
        body (SyncedInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SyncedSaved]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,
body=body,

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
    body: SyncedInput,

) -> Problem | SyncedSaved | None:
    """ Change a synced block (the Header, the Footer or a named block)

     Drafts and scheduled emails holding the block get the change as a new version; emails in an
    approval, sending, paused or sent keep theirs. A named block can be renamed with it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (str): header, footer, or a named block's key
        body (SyncedInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SyncedSaved
     """


    return (await asyncio_detailed(
        workspace=workspace,
kind=kind,
client=client,
body=body,

    )).parsed
