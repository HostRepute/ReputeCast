from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_email_language import AutomationEmailLanguage
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    language: str,
    *,
    version: int,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["version"] = version


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/automation-emails/{id}/languages/{language}".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),language=quote(str(language), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationEmailLanguage | Problem:
    if response.status_code == 200:
        response_200 = AutomationEmailLanguage.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationEmailLanguage | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    version: int,

) -> Response[AutomationEmailLanguage | Problem]:
    """ One language version of an automation email version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationEmailLanguage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
language=language,
version=version,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    version: int,

) -> AutomationEmailLanguage | Problem | None:
    """ One language version of an automation email version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationEmailLanguage | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
language=language,
client=client,
version=version,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    version: int,

) -> Response[AutomationEmailLanguage | Problem]:
    """ One language version of an automation email version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationEmailLanguage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
language=language,
version=version,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    version: int,

) -> AutomationEmailLanguage | Problem | None:
    """ One language version of an automation email version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        version (int): The email version a step sends

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationEmailLanguage | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
language=language,
client=client,
version=version,

    )).parsed
