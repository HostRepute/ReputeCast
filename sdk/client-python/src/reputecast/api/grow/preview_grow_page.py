from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.preview_grow_page_page import PreviewGrowPagePage
from ...models.problem import Problem
from ...models.public_page_preview import PublicPagePreview
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    page: PreviewGrowPagePage,
    *,
    program: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["program"] = program


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/previews/{page}".format(workspace=quote(str(workspace), safe=""),page=quote(str(page), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | PublicPagePreview:
    if response.status_code == 200:
        response_200 = PublicPagePreview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | PublicPagePreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    page: PreviewGrowPagePage,
    *,
    client: AuthenticatedClient,
    program: str | Unset = UNSET,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a Grow public page: the invite page, a referrer's hub, the leaderboard or the
    recommendations page

     Rendered for the referrer with the most signups. Inert: links and buttons don't work and nothing is
    recorded.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewGrowPagePage):
        program (str | Unset): The program (the first one by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
program=program,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    page: PreviewGrowPagePage,
    *,
    client: AuthenticatedClient,
    program: str | Unset = UNSET,

) -> Problem | PublicPagePreview | None:
    """ Preview a Grow public page: the invite page, a referrer's hub, the leaderboard or the
    recommendations page

     Rendered for the referrer with the most signups. Inert: links and buttons don't work and nothing is
    recorded.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewGrowPagePage):
        program (str | Unset): The program (the first one by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPagePreview
     """


    return sync_detailed(
        workspace=workspace,
page=page,
client=client,
program=program,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    page: PreviewGrowPagePage,
    *,
    client: AuthenticatedClient,
    program: str | Unset = UNSET,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a Grow public page: the invite page, a referrer's hub, the leaderboard or the
    recommendations page

     Rendered for the referrer with the most signups. Inert: links and buttons don't work and nothing is
    recorded.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewGrowPagePage):
        program (str | Unset): The program (the first one by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
program=program,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    page: PreviewGrowPagePage,
    *,
    client: AuthenticatedClient,
    program: str | Unset = UNSET,

) -> Problem | PublicPagePreview | None:
    """ Preview a Grow public page: the invite page, a referrer's hub, the leaderboard or the
    recommendations page

     Rendered for the referrer with the most signups. Inert: links and buttons don't work and nothing is
    recorded.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewGrowPagePage):
        program (str | Unset): The program (the first one by default)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPagePreview
     """


    return (await asyncio_detailed(
        workspace=workspace,
page=page,
client=client,
program=program,

    )).parsed
