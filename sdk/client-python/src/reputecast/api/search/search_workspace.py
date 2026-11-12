from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.search_results import SearchResults
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/search".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SearchResults:
    if response.status_code == 200:
        response_200 = SearchResults.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SearchResults]:
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
    q: str | Unset = UNSET,

) -> Response[Problem | SearchResults]:
    """ Search campaigns, contacts, automations, lists, segments, forms and pages by name (the command
    palette)

     Five results a kind. Kinds the caller's role can't open aren't searched; a member scoped to lists
    finds only their contacts and lists.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): At least two characters

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SearchResults]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Problem | SearchResults | None:
    """ Search campaigns, contacts, automations, lists, segments, forms and pages by name (the command
    palette)

     Five results a kind. Kinds the caller's role can't open aren't searched; a member scoped to lists
    finds only their contacts and lists.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): At least two characters

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SearchResults
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Response[Problem | SearchResults]:
    """ Search campaigns, contacts, automations, lists, segments, forms and pages by name (the command
    palette)

     Five results a kind. Kinds the caller's role can't open aren't searched; a member scoped to lists
    finds only their contacts and lists.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): At least two characters

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SearchResults]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Problem | SearchResults | None:
    """ Search campaigns, contacts, automations, lists, segments, forms and pages by name (the command
    palette)

     Five results a kind. Kinds the caller's role can't open aren't searched; a member scoped to lists
    finds only their contacts and lists.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): At least two characters

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SearchResults
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,

    )).parsed
