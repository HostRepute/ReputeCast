from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.import_language_column import ImportLanguageColumn
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    field: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/imports/{id}/fields/{field}/languages".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),field=quote(str(field), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ImportLanguageColumn | Problem:
    if response.status_code == 200:
        response_200 = ImportLanguageColumn.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ImportLanguageColumn | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    field: str,
    *,
    client: AuthenticatedClient,

) -> Response[ImportLanguageColumn | Problem]:
    """ What a source field becomes as the contact's language, and the values that aren't languages

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        field (str): The source field's key

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportLanguageColumn | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
field=field,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    field: str,
    *,
    client: AuthenticatedClient,

) -> ImportLanguageColumn | Problem | None:
    """ What a source field becomes as the contact's language, and the values that aren't languages

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        field (str): The source field's key

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportLanguageColumn | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
field=field,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    field: str,
    *,
    client: AuthenticatedClient,

) -> Response[ImportLanguageColumn | Problem]:
    """ What a source field becomes as the contact's language, and the values that aren't languages

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        field (str): The source field's key

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportLanguageColumn | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
field=field,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    field: str,
    *,
    client: AuthenticatedClient,

) -> ImportLanguageColumn | Problem | None:
    """ What a source field becomes as the contact's language, and the values that aren't languages

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        field (str): The source field's key

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportLanguageColumn | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
field=field,
client=client,

    )).parsed
