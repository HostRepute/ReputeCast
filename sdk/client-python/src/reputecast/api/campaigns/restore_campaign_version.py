from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.composer_restore_version import ComposerRestoreVersion
from ...models.composer_saved_content import ComposerSavedContent
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    body: ComposerRestoreVersion,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/versions/{version}/restore".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),version=quote(str(version), safe=""),),
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
    version: int,
    *,
    client: AuthenticatedClient,
    body: ComposerRestoreVersion,

) -> Response[ComposerSavedContent | Problem]:
    """ Restore a version (saved as a new version)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):
        body (ComposerRestoreVersion):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerSavedContent | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
version=version,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    client: AuthenticatedClient,
    body: ComposerRestoreVersion,

) -> ComposerSavedContent | Problem | None:
    """ Restore a version (saved as a new version)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):
        body (ComposerRestoreVersion):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerSavedContent | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
version=version,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    client: AuthenticatedClient,
    body: ComposerRestoreVersion,

) -> Response[ComposerSavedContent | Problem]:
    """ Restore a version (saved as a new version)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):
        body (ComposerRestoreVersion):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerSavedContent | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
version=version,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    version: int,
    *,
    client: AuthenticatedClient,
    body: ComposerRestoreVersion,

) -> ComposerSavedContent | Problem | None:
    """ Restore a version (saved as a new version)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        version (int):
        body (ComposerRestoreVersion):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerSavedContent | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
version=version,
client=client,
body=body,

    )).parsed
