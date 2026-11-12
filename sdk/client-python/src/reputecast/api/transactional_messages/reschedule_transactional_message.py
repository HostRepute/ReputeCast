from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_reschedule_input import TransactionalRescheduleInput
from ...models.transactional_schedule import TransactionalSchedule
from typing import cast



def _get_kwargs(
    workspace: str,
    message_id: str,
    *,
    body: TransactionalRescheduleInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/transactional/messages/{message_id}/reschedule".format(workspace=quote(str(workspace), safe=""),message_id=quote(str(message_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalSchedule:
    if response.status_code == 200:
        response_200 = TransactionalSchedule.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalSchedule]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalRescheduleInput,

) -> Response[Problem | TransactionalSchedule]:
    """ Move a scheduled message to another send time

     Up to 72 hours from now, like send_at. 409 not_scheduled once it is on its way. Audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered
        body (TransactionalRescheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalSchedule]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
message_id=message_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalRescheduleInput,

) -> Problem | TransactionalSchedule | None:
    """ Move a scheduled message to another send time

     Up to 72 hours from now, like send_at. 409 not_scheduled once it is on its way. Audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered
        body (TransactionalRescheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalSchedule
     """


    return sync_detailed(
        workspace=workspace,
message_id=message_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalRescheduleInput,

) -> Response[Problem | TransactionalSchedule]:
    """ Move a scheduled message to another send time

     Up to 72 hours from now, like send_at. 409 not_scheduled once it is on its way. Audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered
        body (TransactionalRescheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalSchedule]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
message_id=message_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalRescheduleInput,

) -> Problem | TransactionalSchedule | None:
    """ Move a scheduled message to another send time

     Up to 72 hours from now, like send_at. 409 not_scheduled once it is on its way. Audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered
        body (TransactionalRescheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalSchedule
     """


    return (await asyncio_detailed(
        workspace=workspace,
message_id=message_id,
client=client,
body=body,

    )).parsed
