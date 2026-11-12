from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.synced_block import SyncedBlock
from ...models.synced_create_input import SyncedCreateInput
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: SyncedCreateInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/brand-kit/synced".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SyncedBlock:
    if response.status_code == 200:
        response_200 = SyncedBlock.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SyncedBlock]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SyncedCreateInput,

) -> Response[Problem | SyncedBlock]:
    """ Save a section as a named synced block

     The block comes back marked with its key (rc-synced): the composer puts it in place of the section
    it came from. A section that is, or holds, a synced block already is refused.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SyncedCreateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SyncedBlock]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SyncedCreateInput,

) -> Problem | SyncedBlock | None:
    """ Save a section as a named synced block

     The block comes back marked with its key (rc-synced): the composer puts it in place of the section
    it came from. A section that is, or holds, a synced block already is refused.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SyncedCreateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SyncedBlock
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SyncedCreateInput,

) -> Response[Problem | SyncedBlock]:
    """ Save a section as a named synced block

     The block comes back marked with its key (rc-synced): the composer puts it in place of the section
    it came from. A section that is, or holds, a synced block already is refused.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SyncedCreateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SyncedBlock]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SyncedCreateInput,

) -> Problem | SyncedBlock | None:
    """ Save a section as a named synced block

     The block comes back marked with its key (rc-synced): the composer puts it in place of the section
    it came from. A section that is, or holds, a synced block already is refused.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SyncedCreateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SyncedBlock
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
