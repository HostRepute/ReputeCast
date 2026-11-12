from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sms_connect_input import SmsConnectInput
from ...models.sms_connect_result import SmsConnectResult
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: SmsConnectInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/sms/providers".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsConnectResult:
    if response.status_code == 201:
        response_201 = SmsConnectResult.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsConnectResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SmsConnectInput,

) -> Response[Problem | SmsConnectResult]:
    """ Connect an SMS provider (logged)

     Checks the credentials, registers the receipt and inbound webhooks where the API allows it, and
    lists the numbers in the account. Credentials are sealed and never returned.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsConnectResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SmsConnectInput,

) -> Problem | SmsConnectResult | None:
    """ Connect an SMS provider (logged)

     Checks the credentials, registers the receipt and inbound webhooks where the API allows it, and
    lists the numbers in the account. Credentials are sealed and never returned.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsConnectResult
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SmsConnectInput,

) -> Response[Problem | SmsConnectResult]:
    """ Connect an SMS provider (logged)

     Checks the credentials, registers the receipt and inbound webhooks where the API allows it, and
    lists the numbers in the account. Credentials are sealed and never returned.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsConnectResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: SmsConnectInput,

) -> Problem | SmsConnectResult | None:
    """ Connect an SMS provider (logged)

     Checks the credentials, registers the receipt and inbound webhooks where the API allows it, and
    lists the numbers in the account. Credentials are sealed and never returned.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsConnectInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsConnectResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
