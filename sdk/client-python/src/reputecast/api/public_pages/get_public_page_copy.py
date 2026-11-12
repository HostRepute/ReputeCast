from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.public_page_copy import PublicPageCopy
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    language: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["language"] = language


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/public-pages/copy".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | PublicPageCopy:
    if response.status_code == 200:
        response_200 = PublicPageCopy.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | PublicPageCopy]:
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
    language: str | Unset = UNSET,

) -> Response[Problem | PublicPageCopy]:
    """ Get the public pages' copy in every language

     Each workspace language's status (Live, Incomplete with the strings still to write, or Not set up)
    and, with ?language=, that language's strings next to the pages' own.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str | Unset): Also return this language's strings

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPageCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
language=language,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> Problem | PublicPageCopy | None:
    """ Get the public pages' copy in every language

     Each workspace language's status (Live, Incomplete with the strings still to write, or Not set up)
    and, with ?language=, that language's strings next to the pages' own.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str | Unset): Also return this language's strings

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPageCopy
     """


    return sync_detailed(
        workspace=workspace,
client=client,
language=language,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> Response[Problem | PublicPageCopy]:
    """ Get the public pages' copy in every language

     Each workspace language's status (Live, Incomplete with the strings still to write, or Not set up)
    and, with ?language=, that language's strings next to the pages' own.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str | Unset): Also return this language's strings

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPageCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
language=language,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> Problem | PublicPageCopy | None:
    """ Get the public pages' copy in every language

     Each workspace language's status (Live, Incomplete with the strings still to write, or Not set up)
    and, with ?language=, that language's strings next to the pages' own.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str | Unset): Also return this language's strings

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPageCopy
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
language=language,

    )).parsed
