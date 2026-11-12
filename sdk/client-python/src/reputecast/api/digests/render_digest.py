from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.digest_render import DigestRender
from ...models.digest_render_input import DigestRenderInput
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    digest_id: UUID,
    *,
    body: DigestRenderInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/digests/{digest_id}/render".format(workspace=quote(str(workspace), safe=""),digest_id=quote(str(digest_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DigestRender | Problem:
    if response.status_code == 200:
        response_200 = DigestRender.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DigestRender | Problem]:
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
    body: DigestRenderInput,

) -> Response[DigestRender | Problem]:
    """ Draw the Feed items block and the subject with a preview's posts (the composer in digest mode)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestRender | Problem]
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
    body: DigestRenderInput,

) -> DigestRender | Problem | None:
    """ Draw the Feed items block and the subject with a preview's posts (the composer in digest mode)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestRender | Problem
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
    body: DigestRenderInput,

) -> Response[DigestRender | Problem]:
    """ Draw the Feed items block and the subject with a preview's posts (the composer in digest mode)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestRender | Problem]
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
    body: DigestRenderInput,

) -> DigestRender | Problem | None:
    """ Draw the Feed items block and the subject with a preview's posts (the composer in digest mode)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestRender | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,
body=body,

    )).parsed
