from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.archive_issue_page import ArchiveIssuePage
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q

    params["list"] = list_

    params["page"] = page


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/public-pages/archive/issues".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ArchiveIssuePage | Problem:
    if response.status_code == 200:
        response_200 = ArchiveIssuePage.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ArchiveIssuePage | Problem]:
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
    list_: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> Response[ArchiveIssuePage | Problem]:
    """ List the archive's issues, hidden ones included

     Sent campaigns of the archived lists are filed the first time the archive is read after they finish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search titles, subjects and excerpts
        list_ (str | Unset): Only issues filed under this list (id)
        page (int | Unset): Page, from 1 (10 issues a page)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArchiveIssuePage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
list_=list_,
page=page,

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
    list_: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> ArchiveIssuePage | Problem | None:
    """ List the archive's issues, hidden ones included

     Sent campaigns of the archived lists are filed the first time the archive is read after they finish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search titles, subjects and excerpts
        list_ (str | Unset): Only issues filed under this list (id)
        page (int | Unset): Page, from 1 (10 issues a page)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArchiveIssuePage | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,
list_=list_,
page=page,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> Response[ArchiveIssuePage | Problem]:
    """ List the archive's issues, hidden ones included

     Sent campaigns of the archived lists are filed the first time the archive is read after they finish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search titles, subjects and excerpts
        list_ (str | Unset): Only issues filed under this list (id)
        page (int | Unset): Page, from 1 (10 issues a page)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArchiveIssuePage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
list_=list_,
page=page,

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
    list_: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> ArchiveIssuePage | Problem | None:
    """ List the archive's issues, hidden ones included

     Sent campaigns of the archived lists are filed the first time the archive is read after they finish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search titles, subjects and excerpts
        list_ (str | Unset): Only issues filed under this list (id)
        page (int | Unset): Page, from 1 (10 issues a page)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArchiveIssuePage | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,
list_=list_,
page=page,

    )).parsed
