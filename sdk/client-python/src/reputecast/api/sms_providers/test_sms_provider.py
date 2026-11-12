from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sms_message_view import SmsMessageView
from ...models.sms_provider_test_input import SmsProviderTestInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    provider_id: UUID,
    *,
    body: SmsProviderTestInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/sms/providers/{provider_id}/test".format(workspace=quote(str(workspace), safe=""),provider_id=quote(str(provider_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsMessageView:
    if response.status_code == 200:
        response_200 = SmsMessageView.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsMessageView]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsProviderTestInput,

) -> Response[Problem | SmsMessageView]:
    """ Text a test to a member's phone through this provider (logged)

     The timeline waits for the delivery receipt: poll the message.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsProviderTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsMessageView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsProviderTestInput,

) -> Problem | SmsMessageView | None:
    """ Text a test to a member's phone through this provider (logged)

     The timeline waits for the delivery receipt: poll the message.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsProviderTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsMessageView
     """


    return sync_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsProviderTestInput,

) -> Response[Problem | SmsMessageView]:
    """ Text a test to a member's phone through this provider (logged)

     The timeline waits for the delivery receipt: poll the message.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsProviderTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsMessageView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsProviderTestInput,

) -> Problem | SmsMessageView | None:
    """ Text a test to a member's phone through this provider (logged)

     The timeline waits for the delivery receipt: poll the message.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsProviderTestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsMessageView
     """


    return (await asyncio_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,
body=body,

    )).parsed
