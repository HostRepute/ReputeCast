from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.digest_issues import DigestIssues
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    digest_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/digests/{digest_id}/issues".format(workspace=quote(str(workspace), safe=""),digest_id=quote(str(digest_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DigestIssues | Problem:
    if response.status_code == 200:
        response_200 = DigestIssues.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DigestIssues | Problem]:
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

) -> Response[DigestIssues | Problem]:
    """ A digest's issues: sent, skipped, held, waiting and paused, each with its numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestIssues | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,

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

) -> DigestIssues | Problem | None:
    """ A digest's issues: sent, skipped, held, waiting and paused, each with its numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestIssues | Problem
     """


    return sync_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    digest_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[DigestIssues | Problem]:
    """ A digest's issues: sent, skipped, held, waiting and paused, each with its numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestIssues | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
digest_id=digest_id,

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

) -> DigestIssues | Problem | None:
    """ A digest's issues: sent, skipped, held, waiting and paused, each with its numbers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        digest_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestIssues | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
digest_id=digest_id,
client=client,

    )).parsed
