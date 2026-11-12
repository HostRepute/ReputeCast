from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.grow_preview_contacts import GrowPreviewContacts
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    *,
    program: UUID | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_program: str | Unset = UNSET
    if not isinstance(program, Unset):
        json_program = str(program)
    params["program"] = json_program


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/blocks/preview-contacts".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> GrowPreviewContacts | Problem:
    if response.status_code == 200:
        response_200 = GrowPreviewContacts.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[GrowPreviewContacts | Problem]:
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
    program: UUID | Unset = UNSET,

) -> Response[GrowPreviewContacts | Problem]:
    """ Readers to preview the Grow blocks as: referrers, and readers the block hides for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GrowPreviewContacts | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    program: UUID | Unset = UNSET,

) -> GrowPreviewContacts | Problem | None:
    """ Readers to preview the Grow blocks as: referrers, and readers the block hides for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GrowPreviewContacts | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
program=program,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    program: UUID | Unset = UNSET,

) -> Response[GrowPreviewContacts | Problem]:
    """ Readers to preview the Grow blocks as: referrers, and readers the block hides for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GrowPreviewContacts | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    program: UUID | Unset = UNSET,

) -> GrowPreviewContacts | Problem | None:
    """ Readers to preview the Grow blocks as: referrers, and readers the block hides for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GrowPreviewContacts | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
program=program,

    )).parsed
