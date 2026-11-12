from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.import_field_mapping import ImportFieldMapping
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    guesses: bool | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["guesses"] = guesses


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/imports/{id}/fields".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ImportFieldMapping | Problem:
    if response.status_code == 200:
        response_200 = ImportFieldMapping.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ImportFieldMapping | Problem]:
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
    guesses: bool | Unset = UNSET,

) -> Response[ImportFieldMapping | Problem]:
    """ Source fields with sample values and their mapping

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        guesses (bool | Unset): Guess every target again instead of returning the saved mapping

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportFieldMapping | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
guesses=guesses,

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
    guesses: bool | Unset = UNSET,

) -> ImportFieldMapping | Problem | None:
    """ Source fields with sample values and their mapping

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        guesses (bool | Unset): Guess every target again instead of returning the saved mapping

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportFieldMapping | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
guesses=guesses,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    guesses: bool | Unset = UNSET,

) -> Response[ImportFieldMapping | Problem]:
    """ Source fields with sample values and their mapping

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        guesses (bool | Unset): Guess every target again instead of returning the saved mapping

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportFieldMapping | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
guesses=guesses,

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
    guesses: bool | Unset = UNSET,

) -> ImportFieldMapping | Problem | None:
    """ Source fields with sample values and their mapping

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        guesses (bool | Unset): Guess every target again instead of returning the saved mapping

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportFieldMapping | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
guesses=guesses,

    )).parsed
