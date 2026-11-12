from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.preview_public_page_page import PreviewPublicPagePage
from ...models.problem import Problem
from ...models.public_page_preview import PublicPagePreview
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    page: PreviewPublicPagePage,
    *,
    form: str | Unset = UNSET,
    draft: str | Unset = UNSET,
    contact: str | Unset = UNSET,
    language: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["form"] = form

    params["draft"] = draft

    params["contact"] = contact

    params["language"] = language


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/public-pages/preview/{page}".format(workspace=quote(str(workspace), safe=""),page=quote(str(page), safe=""),),
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
    page: PreviewPublicPagePage,
    *,
    client: AuthenticatedClient,
    form: str | Unset = UNSET,
    draft: str | Unset = UNSET,
    contact: str | Unset = UNSET,
    language: str | Unset = UNSET,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a public page as a subscriber sees it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewPublicPagePage):
        form (str | Unset): Show a form's hosted page (drafts too) instead of the subscribe page
        draft (str | Unset): Unsaved page options to show: the body of save-public-pages, as JSON
        contact (str | Unset): Preview the pages of this subscriber
        language (str | Unset): Preview the page in this language

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
form=form,
draft=draft,
contact=contact,
language=language,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    page: PreviewPublicPagePage,
    *,
    client: AuthenticatedClient,
    form: str | Unset = UNSET,
    draft: str | Unset = UNSET,
    contact: str | Unset = UNSET,
    language: str | Unset = UNSET,

) -> Problem | PublicPagePreview | None:
    """ Preview a public page as a subscriber sees it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewPublicPagePage):
        form (str | Unset): Show a form's hosted page (drafts too) instead of the subscribe page
        draft (str | Unset): Unsaved page options to show: the body of save-public-pages, as JSON
        contact (str | Unset): Preview the pages of this subscriber
        language (str | Unset): Preview the page in this language

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
form=form,
draft=draft,
contact=contact,
language=language,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    page: PreviewPublicPagePage,
    *,
    client: AuthenticatedClient,
    form: str | Unset = UNSET,
    draft: str | Unset = UNSET,
    contact: str | Unset = UNSET,
    language: str | Unset = UNSET,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a public page as a subscriber sees it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewPublicPagePage):
        form (str | Unset): Show a form's hosted page (drafts too) instead of the subscribe page
        draft (str | Unset): Unsaved page options to show: the body of save-public-pages, as JSON
        contact (str | Unset): Preview the pages of this subscriber
        language (str | Unset): Preview the page in this language

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
page=page,
form=form,
draft=draft,
contact=contact,
language=language,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    page: PreviewPublicPagePage,
    *,
    client: AuthenticatedClient,
    form: str | Unset = UNSET,
    draft: str | Unset = UNSET,
    contact: str | Unset = UNSET,
    language: str | Unset = UNSET,

) -> Problem | PublicPagePreview | None:
    """ Preview a public page as a subscriber sees it

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        page (PreviewPublicPagePage):
        form (str | Unset): Show a form's hosted page (drafts too) instead of the subscribe page
        draft (str | Unset): Unsaved page options to show: the body of save-public-pages, as JSON
        contact (str | Unset): Preview the pages of this subscriber
        language (str | Unset): Preview the page in this language

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
form=form,
draft=draft,
contact=contact,
language=language,

    )).parsed
