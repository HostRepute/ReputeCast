from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sms_registration_panel import SmsRegistrationPanel
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    sender_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/sms/senders/{sender_id}/registration".format(workspace=quote(str(workspace), safe=""),sender_id=quote(str(sender_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsRegistrationPanel:
    if response.status_code == 200:
        response_200 = SmsRegistrationPanel.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsRegistrationPanel]:
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

) -> Response[Problem | SmsRegistrationPanel]:
    """ A sender's registrations, and what the provider's form asks for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsRegistrationPanel]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
sender_id=sender_id,

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

) -> Problem | SmsRegistrationPanel | None:
    """ A sender's registrations, and what the provider's form asks for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsRegistrationPanel
     """


    return sync_detailed(
        workspace=workspace,
sender_id=sender_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    sender_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SmsRegistrationPanel]:
    """ A sender's registrations, and what the provider's form asks for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsRegistrationPanel]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
sender_id=sender_id,

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

) -> Problem | SmsRegistrationPanel | None:
    """ A sender's registrations, and what the provider's form asks for

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        sender_id (UUID): Sender id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsRegistrationPanel
     """


    return (await asyncio_detailed(
        workspace=workspace,
sender_id=sender_id,
client=client,

    )).parsed
