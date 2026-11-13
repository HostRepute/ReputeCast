from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_website_comments_status import ListWebsiteCommentsStatus
from ...models.problem import Problem
from ...models.website_comment_list import WebsiteCommentList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    status: ListWebsiteCommentsStatus | Unset = UNSET,
    post: str | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["post"] = post

    params["q"] = q

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/website/comments".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebsiteCommentList:
    if response.status_code == 200:
        response_200 = WebsiteCommentList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebsiteCommentList]:
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
    status: ListWebsiteCommentsStatus | Unset = UNSET,
    post: str | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[Problem | WebsiteCommentList]:
    """ Website › Comments: comments waiting for review, published or hidden

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsiteCommentsStatus | Unset): The tab: waiting for review (the default),
            published or hidden
        post (str | Unset): A post's id
        q (str | Unset): Words in the comment, the commenter's name or address
        cursor (str | Unset): nextCursor of the page before
        limit (int | Unset): Comments per page (50 by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteCommentList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
post=post,
q=q,
cursor=cursor,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListWebsiteCommentsStatus | Unset = UNSET,
    post: str | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Problem | WebsiteCommentList | None:
    """ Website › Comments: comments waiting for review, published or hidden

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsiteCommentsStatus | Unset): The tab: waiting for review (the default),
            published or hidden
        post (str | Unset): A post's id
        q (str | Unset): Words in the comment, the commenter's name or address
        cursor (str | Unset): nextCursor of the page before
        limit (int | Unset): Comments per page (50 by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteCommentList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,
post=post,
q=q,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListWebsiteCommentsStatus | Unset = UNSET,
    post: str | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[Problem | WebsiteCommentList]:
    """ Website › Comments: comments waiting for review, published or hidden

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsiteCommentsStatus | Unset): The tab: waiting for review (the default),
            published or hidden
        post (str | Unset): A post's id
        q (str | Unset): Words in the comment, the commenter's name or address
        cursor (str | Unset): nextCursor of the page before
        limit (int | Unset): Comments per page (50 by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteCommentList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
post=post,
q=q,
cursor=cursor,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListWebsiteCommentsStatus | Unset = UNSET,
    post: str | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Problem | WebsiteCommentList | None:
    """ Website › Comments: comments waiting for review, published or hidden

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsiteCommentsStatus | Unset): The tab: waiting for review (the default),
            published or hidden
        post (str | Unset): A post's id
        q (str | Unset): Words in the comment, the commenter's name or address
        cursor (str | Unset): nextCursor of the page before
        limit (int | Unset): Comments per page (50 by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteCommentList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,
post=post,
q=q,
cursor=cursor,
limit=limit,

    )).parsed
