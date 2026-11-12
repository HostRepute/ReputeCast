from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.website_sent_as_campaign import WebsiteSentAsCampaign
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    post: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/website/posts/{post}/send-as-campaign".format(workspace=quote(str(workspace), safe=""),post=quote(str(post), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WebsiteSentAsCampaign:
    if response.status_code == 201:
        response_201 = WebsiteSentAsCampaign.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WebsiteSentAsCampaign]:
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

) -> Response[Problem | WebsiteSentAsCampaign]:
    """ Make a campaign draft of a web-only post

     Its title becomes the subject, its excerpt the preheader and its blocks the email. Pick the audience
    in the composer.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteSentAsCampaign]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
post=post,

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

) -> Problem | WebsiteSentAsCampaign | None:
    """ Make a campaign draft of a web-only post

     Its title becomes the subject, its excerpt the preheader and its blocks the email. Pick the audience
    in the composer.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteSentAsCampaign
     """


    return sync_detailed(
        workspace=workspace,
post=post,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    post: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | WebsiteSentAsCampaign]:
    """ Make a campaign draft of a web-only post

     Its title becomes the subject, its excerpt the preheader and its blocks the email. Pick the audience
    in the composer.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WebsiteSentAsCampaign]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
post=post,

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

) -> Problem | WebsiteSentAsCampaign | None:
    """ Make a campaign draft of a web-only post

     Its title becomes the subject, its excerpt the preheader and its blocks the email. Pick the audience
    in the composer.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        post (UUID): The post's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WebsiteSentAsCampaign
     """


    return (await asyncio_detailed(
        workspace=workspace,
post=post,
client=client,

    )).parsed
