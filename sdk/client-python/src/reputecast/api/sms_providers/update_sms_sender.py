from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sms_sender_rows import SmsSenderRows
from ...models.sms_sender_update import SmsSenderUpdate
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    sender_id: UUID,
    *,
    body: SmsSenderUpdate,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/workspaces/{workspace}/sms/senders/{sender_id}".format(workspace=quote(str(workspace), safe=""),sender_id=quote(str(sender_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsSenderRows:
    if response.status_code == 200:
        response_200 = SmsSenderRows.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsSenderRows]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    sender_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsSenderUpdate,

) -> Response[Problem | SmsSenderRows]:
    """ Change a sender's countries (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id
        body (SmsSenderUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsSenderRows]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
sender_id=sender_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    sender_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsSenderUpdate,

) -> Problem | SmsSenderRows | None:
    """ Change a sender's countries (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id
        body (SmsSenderUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsSenderRows
     """


    return sync_detailed(
        workspace=workspace,
sender_id=sender_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    sender_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsSenderUpdate,

) -> Response[Problem | SmsSenderRows]:
    """ Change a sender's countries (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id
        body (SmsSenderUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsSenderRows]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
sender_id=sender_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    sender_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsSenderUpdate,

) -> Problem | SmsSenderRows | None:
    """ Change a sender's countries (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id
        body (SmsSenderUpdate):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsSenderRows
     """


    return (await asyncio_detailed(
        workspace=workspace,
sender_id=sender_id,
client=client,
body=body,

    )).parsed
