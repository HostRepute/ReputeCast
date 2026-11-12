from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.composer_test_send_result import ComposerTestSendResult
from ...models.digest_test_input import DigestTestInput
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    digest_id: UUID,
    *,
    body: DigestTestInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/digests/{digest_id}/test".format(workspace=quote(str(workspace), safe=""),digest_id=quote(str(digest_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ComposerTestSendResult | Problem:
    if response.status_code == 200:
        response_200 = ComposerTestSendResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ComposerTestSendResult | Problem]:
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
    body: DigestTestInput,

) -> Response[ComposerTestSendResult | Problem]:
    """ Send the next issue, as it would be built now, as a test

     Goes through the System route. Without campaigns:send, test emails can only go to your own address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerTestSendResult | Problem]
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
    body: DigestTestInput,

) -> ComposerTestSendResult | Problem | None:
    """ Send the next issue, as it would be built now, as a test

     Goes through the System route. Without campaigns:send, test emails can only go to your own address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerTestSendResult | Problem
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
    body: DigestTestInput,

) -> Response[ComposerTestSendResult | Problem]:
    """ Send the next issue, as it would be built now, as a test

     Goes through the System route. Without campaigns:send, test emails can only go to your own address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComposerTestSendResult | Problem]
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
    body: DigestTestInput,

) -> ComposerTestSendResult | Problem | None:
    """ Send the next issue, as it would be built now, as a test

     Goes through the System route. Without campaigns:send, test emails can only go to your own address.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):
        body (DigestTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComposerTestSendResult | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,
body=body,

    )).parsed
