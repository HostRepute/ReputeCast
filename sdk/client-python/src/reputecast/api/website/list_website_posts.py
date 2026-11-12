from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_website_posts_source import ListWebsitePostsSource
from ...models.list_website_posts_status import ListWebsitePostsStatus
from ...models.list_website_posts_visibility import ListWebsitePostsVisibility
from ...models.problem import Problem
from ...models.website_post_list import WebsitePostList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    status: ListWebsitePostsStatus | Unset = UNSET,
    visibility: ListWebsitePostsVisibility | Unset = UNSET,
    tag: str | Unset = UNSET,
    source: ListWebsitePostsSource | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    json_visibility: str | Unset = UNSET
    if not isinstance(visibility, Unset):
        json_visibility = visibility.value

    params["visibility"] = json_visibility

    params["tag"] = tag

    json_source: str | Unset = UNSET
    if not isinstance(source, Unset):
        json_source = source.value

    params["source"] = json_source

    params["q"] = q

    params["page"] = page

    params["pageSize"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/website/posts".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebsitePostList:
    if response.status_code == 200:
        response_200 = WebsitePostList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebsitePostList]:
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
    status: ListWebsitePostsStatus | Unset = UNSET,
    visibility: ListWebsitePostsVisibility | Unset = UNSET,
    tag: str | Unset = UNSET,
    source: ListWebsitePostsSource | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,

) -> Response[Problem | WebsitePostList]:
    """ Website › Posts, with their 30-day numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsitePostsStatus | Unset): A status (all by default)
        visibility (ListWebsitePostsVisibility | Unset):
        tag (str | Unset): A tag's id
        source (ListWebsitePostsSource | Unset):
        q (str | Unset): Search by title
        page (int | Unset):
        page_size (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsitePostList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
visibility=visibility,
tag=tag,
source=source,
q=q,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListWebsitePostsStatus | Unset = UNSET,
    visibility: ListWebsitePostsVisibility | Unset = UNSET,
    tag: str | Unset = UNSET,
    source: ListWebsitePostsSource | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,

) -> Problem | WebsitePostList | None:
    """ Website › Posts, with their 30-day numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsitePostsStatus | Unset): A status (all by default)
        visibility (ListWebsitePostsVisibility | Unset):
        tag (str | Unset): A tag's id
        source (ListWebsitePostsSource | Unset):
        q (str | Unset): Search by title
        page (int | Unset):
        page_size (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsitePostList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,
visibility=visibility,
tag=tag,
source=source,
q=q,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListWebsitePostsStatus | Unset = UNSET,
    visibility: ListWebsitePostsVisibility | Unset = UNSET,
    tag: str | Unset = UNSET,
    source: ListWebsitePostsSource | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,

) -> Response[Problem | WebsitePostList]:
    """ Website › Posts, with their 30-day numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsitePostsStatus | Unset): A status (all by default)
        visibility (ListWebsitePostsVisibility | Unset):
        tag (str | Unset): A tag's id
        source (ListWebsitePostsSource | Unset):
        q (str | Unset): Search by title
        page (int | Unset):
        page_size (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsitePostList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
visibility=visibility,
tag=tag,
source=source,
q=q,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListWebsitePostsStatus | Unset = UNSET,
    visibility: ListWebsitePostsVisibility | Unset = UNSET,
    tag: str | Unset = UNSET,
    source: ListWebsitePostsSource | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,

) -> Problem | WebsitePostList | None:
    """ Website › Posts, with their 30-day numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListWebsitePostsStatus | Unset): A status (all by default)
        visibility (ListWebsitePostsVisibility | Unset):
        tag (str | Unset): A tag's id
        source (ListWebsitePostsSource | Unset):
        q (str | Unset): Search by title
        page (int | Unset):
        page_size (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsitePostList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,
visibility=visibility,
tag=tag,
source=source,
q=q,
page=page,
page_size=page_size,

    )).parsed
