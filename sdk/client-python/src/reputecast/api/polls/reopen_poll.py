from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.poll_results import PollResults
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    poll_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/polls/{poll_id}/reopen".format(workspace=quote(str(workspace), safe=""),poll_id=quote(str(poll_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PollResults | Problem:
    if response.status_code == 200:
        response_200 = PollResults.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PollResults | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PollResults | Problem]:
    """ Reopen a closed poll (logged)

     Answers made after reopening count as usual; it stays open until someone closes it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PollResults | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
poll_id=poll_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,

) -> PollResults | Problem | None:
    """ Reopen a closed poll (logged)

     Answers made after reopening count as usual; it stays open until someone closes it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PollResults | Problem
     """


    return sync_detailed(
        workspace=workspace,
poll_id=poll_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PollResults | Problem]:
    """ Reopen a closed poll (logged)

     Answers made after reopening count as usual; it stays open until someone closes it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PollResults | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
poll_id=poll_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,

) -> PollResults | Problem | None:
    """ Reopen a closed poll (logged)

     Answers made after reopening count as usual; it stays open until someone closes it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PollResults | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
poll_id=poll_id,
client=client,

    )).parsed
