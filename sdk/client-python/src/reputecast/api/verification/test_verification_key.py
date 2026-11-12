from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.verification_key_input import VerificationKeyInput
from ...models.verification_key_test import VerificationKeyTest
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: VerificationKeyInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/verification/services/test".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VerificationKeyTest:
    if response.status_code == 200:
        response_200 = VerificationKeyTest.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VerificationKeyTest]:
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
    body: VerificationKeyInput,

) -> Response[Problem | VerificationKeyTest]:
    """ Test a service's API key by reading its balance (no address is sent)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (VerificationKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationKeyTest]
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
    body: VerificationKeyInput,

) -> Problem | VerificationKeyTest | None:
    """ Test a service's API key by reading its balance (no address is sent)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (VerificationKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationKeyTest
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
    body: VerificationKeyInput,

) -> Response[Problem | VerificationKeyTest]:
    """ Test a service's API key by reading its balance (no address is sent)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (VerificationKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationKeyTest]
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
    body: VerificationKeyInput,

) -> Problem | VerificationKeyTest | None:
    """ Test a service's API key by reading its balance (no address is sent)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (VerificationKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationKeyTest
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
