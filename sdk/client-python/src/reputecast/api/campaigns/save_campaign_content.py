from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.composer_save_content import ComposerSaveContent
from ...models.composer_saved_content import ComposerSavedContent
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    body: ComposerSaveContent,
    language: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    params: dict[str, Any] = {}

    params["language"] = language


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/content".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
        "params": params,
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ComposerSavedContent | Problem:
    if response.status_code == 200:
        response_200 = ComposerSavedContent.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ComposerSavedContent | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ComposerSaveContent,
    language: str | Unset = UNSET,

) -> Response[ComposerSavedContent | Problem]:
    """ Save the content as a new version (autosave)

     Fails with 409 version_conflict when someone saved a newer version since baseVersion; the problem's
    meta says who, when and which sections each side changed. Send force: true to keep yours on top, or
    stash: true to keep yours in the history under the current version (Load theirs). Saving content
    that is already the current version stores nothing (unchanged: true). Fails with 409
    approval_would_cancel while the campaign has a pending or approved request, since the save cancels
    it: send cancelApproval: true once the person agreed.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str | Unset): Save a language version (its code, such as de); empty saves the
            source
        body (ComposerSaveContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerSavedContent | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
body=body,
language=language,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ComposerSaveContent,
    language: str | Unset = UNSET,

) -> ComposerSavedContent | Problem | None:
    """ Save the content as a new version (autosave)

     Fails with 409 version_conflict when someone saved a newer version since baseVersion; the problem's
    meta says who, when and which sections each side changed. Send force: true to keep yours on top, or
    stash: true to keep yours in the history under the current version (Load theirs). Saving content
    that is already the current version stores nothing (unchanged: true). Fails with 409
    approval_would_cancel while the campaign has a pending or approved request, since the save cancels
    it: send cancelApproval: true once the person agreed.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str | Unset): Save a language version (its code, such as de); empty saves the
            source
        body (ComposerSaveContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerSavedContent | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
body=body,
language=language,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ComposerSaveContent,
    language: str | Unset = UNSET,

) -> Response[ComposerSavedContent | Problem]:
    """ Save the content as a new version (autosave)

     Fails with 409 version_conflict when someone saved a newer version since baseVersion; the problem's
    meta says who, when and which sections each side changed. Send force: true to keep yours on top, or
    stash: true to keep yours in the history under the current version (Load theirs). Saving content
    that is already the current version stores nothing (unchanged: true). Fails with 409
    approval_would_cancel while the campaign has a pending or approved request, since the save cancels
    it: send cancelApproval: true once the person agreed.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str | Unset): Save a language version (its code, such as de); empty saves the
            source
        body (ComposerSaveContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerSavedContent | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
body=body,
language=language,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ComposerSaveContent,
    language: str | Unset = UNSET,

) -> ComposerSavedContent | Problem | None:
    """ Save the content as a new version (autosave)

     Fails with 409 version_conflict when someone saved a newer version since baseVersion; the problem's
    meta says who, when and which sections each side changed. Send force: true to keep yours on top, or
    stash: true to keep yours in the history under the current version (Load theirs). Saving content
    that is already the current version stores nothing (unchanged: true). Fails with 409
    approval_would_cancel while the campaign has a pending or approved request, since the save cancels
    it: send cancelApproval: true once the person agreed.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str | Unset): Save a language version (its code, such as de); empty saves the
            source
        body (ComposerSaveContent):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerSavedContent | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
body=body,
language=language,

    )).parsed
