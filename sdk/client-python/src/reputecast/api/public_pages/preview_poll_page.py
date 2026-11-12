from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.poll_page_preview_input import PollPagePreviewInput
from ...models.problem import Problem
from ...models.public_page_preview import PublicPagePreview
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: PollPagePreviewInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/public-pages/poll-preview".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
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
    *,
    client: AuthenticatedClient,
    body: PollPagePreviewInput,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a poll's answer page as a contact sees it after answering

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (PollPagePreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: PollPagePreviewInput,

) -> Problem | PublicPagePreview | None:
    """ Preview a poll's answer page as a contact sees it after answering

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (PollPagePreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPagePreview
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: PollPagePreviewInput,

) -> Response[Problem | PublicPagePreview]:
    """ Preview a poll's answer page as a contact sees it after answering

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (PollPagePreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPagePreview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: PollPagePreviewInput,

) -> Problem | PublicPagePreview | None:
    """ Preview a poll's answer page as a contact sees it after answering

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (PollPagePreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPagePreview
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
