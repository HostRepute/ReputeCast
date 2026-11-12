from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.digest_detail import DigestDetail
from ...models.digest_pause_input import DigestPauseInput
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    digest_id: UUID,
    *,
    body: DigestPauseInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/digests/{digest_id}/pause".format(workspace=quote(str(workspace), safe=""),digest_id=quote(str(digest_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DigestDetail | Problem:
    if response.status_code == 200:
        response_200 = DigestDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DigestDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    body: DigestPauseInput,

) -> Response[DigestDetail | Problem]:
    """ Pause a digest: no new issues are prepared

     An issue waiting to go out is skipped and its posts stay new; one that is sending pauses like any
    send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestPauseInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    body: DigestPauseInput,

) -> DigestDetail | Problem | None:
    """ Pause a digest: no new issues are prepared

     An issue waiting to go out is skipped and its posts stay new; one that is sending pauses like any
    send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestPauseInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    body: DigestPauseInput,

) -> Response[DigestDetail | Problem]:
    """ Pause a digest: no new issues are prepared

     An issue waiting to go out is skipped and its posts stay new; one that is sending pauses like any
    send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestPauseInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,
    body: DigestPauseInput,

) -> DigestDetail | Problem | None:
    """ Pause a digest: no new issues are prepared

     An issue waiting to go out is skipped and its posts stay new; one that is sending pauses like any
    send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestPauseInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,
body=body,

    )).parsed
