from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.language_removal import LanguageRemoval
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    code: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/languages/{code}/removal".format(workspace=quote(str(workspace), safe=""),code=quote(str(code), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> LanguageRemoval | Problem:
    if response.status_code == 200:
        response_200 = LanguageRemoval.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[LanguageRemoval | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    code: str,
    *,
    client: AuthenticatedClient,

) -> Response[LanguageRemoval | Problem]:
    """ What removing a language affects

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[LanguageRemoval | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
code=code,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    code: str,
    *,
    client: AuthenticatedClient,

) -> LanguageRemoval | Problem | None:
    """ What removing a language affects

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        LanguageRemoval | Problem
     """


    return sync_detailed(
        workspace=workspace,
code=code,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    code: str,
    *,
    client: AuthenticatedClient,

) -> Response[LanguageRemoval | Problem]:
    """ What removing a language affects

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[LanguageRemoval | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
code=code,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    code: str,
    *,
    client: AuthenticatedClient,

) -> LanguageRemoval | Problem | None:
    """ What removing a language affects

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        LanguageRemoval | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
code=code,
client=client,

    )).parsed
