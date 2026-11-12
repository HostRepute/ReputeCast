from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.workspace_languages import WorkspaceLanguages
from typing import cast



def _get_kwargs(
    workspace: str,
    code: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/workspaces/{workspace}/languages/{code}".format(workspace=quote(str(workspace), safe=""),code=quote(str(code), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WorkspaceLanguages:
    if response.status_code == 200:
        response_200 = WorkspaceLanguages.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WorkspaceLanguages]:
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

) -> Response[Problem | WorkspaceLanguages]:
    """ Remove a language from the workspace

     Its email versions and page copy are kept but no longer used; its contacts keep their language and
    get the fallback. The default and the fallback can't be removed (409).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WorkspaceLanguages]
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

) -> Problem | WorkspaceLanguages | None:
    """ Remove a language from the workspace

     Its email versions and page copy are kept but no longer used; its contacts keep their language and
    get the fallback. The default and the fallback can't be removed (409).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WorkspaceLanguages
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

) -> Response[Problem | WorkspaceLanguages]:
    """ Remove a language from the workspace

     Its email versions and page copy are kept but no longer used; its contacts keep their language and
    get the fallback. The default and the fallback can't be removed (409).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WorkspaceLanguages]
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

) -> Problem | WorkspaceLanguages | None:
    """ Remove a language from the workspace

     Its email versions and page copy are kept but no longer used; its contacts keep their language and
    get the fallback. The default and the fallback can't be removed (409).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WorkspaceLanguages
     """


    return (await asyncio_detailed(
        workspace=workspace,
code=code,
client=client,

    )).parsed
