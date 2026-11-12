from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.email_languages import EmailLanguages
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    version: int,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["version"] = version


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/automation-emails/{id}/languages".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmailLanguages | Problem:
    if response.status_code == 200:
        response_200 = EmailLanguages.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EmailLanguages | Problem]:
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
    version: int,

) -> Response[EmailLanguages | Problem]:
    """ An automation email version's language versions with their status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
version=version,

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
    version: int,

) -> EmailLanguages | Problem | None:
    """ An automation email version's language versions with their status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
version=version,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    version: int,

) -> Response[EmailLanguages | Problem]:
    """ An automation email version's language versions with their status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
version=version,

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
    version: int,

) -> EmailLanguages | Problem | None:
    """ An automation email version's language versions with their status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
version=version,

    )).parsed
