from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.website_post_content import WebsitePostContent
from ...models.website_post_saved import WebsitePostSaved
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    post: UUID,
    *,
    body: WebsitePostContent,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/website/posts/{post}/content".format(workspace=quote(str(workspace), safe=""),post=quote(str(post), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebsitePostSaved:
    if response.status_code == 200:
        response_200 = WebsitePostSaved.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebsitePostSaved]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    post: UUID,
    *,
    client: AuthenticatedClient,
    body: WebsitePostContent,

) -> Response[Problem | WebsitePostSaved]:
    """ Autosave or save a web-only post's content

     Another save since baseVersion is a conflict (409) unless force is set.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id
        body (WebsitePostContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsitePostSaved]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
post=post,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    post: UUID,
    *,
    client: AuthenticatedClient,
    body: WebsitePostContent,

) -> Problem | WebsitePostSaved | None:
    """ Autosave or save a web-only post's content

     Another save since baseVersion is a conflict (409) unless force is set.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id
        body (WebsitePostContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsitePostSaved
     """


    return sync_detailed(
        workspace=workspace,
post=post,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    post: UUID,
    *,
    client: AuthenticatedClient,
    body: WebsitePostContent,

) -> Response[Problem | WebsitePostSaved]:
    """ Autosave or save a web-only post's content

     Another save since baseVersion is a conflict (409) unless force is set.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id
        body (WebsitePostContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsitePostSaved]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
post=post,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    post: UUID,
    *,
    client: AuthenticatedClient,
    body: WebsitePostContent,

) -> Problem | WebsitePostSaved | None:
    """ Autosave or save a web-only post's content

     Another save since baseVersion is a conflict (409) unless force is set.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id
        body (WebsitePostContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsitePostSaved
     """


    return (await asyncio_detailed(
        workspace=workspace,
post=post,
client=client,
body=body,

    )).parsed
