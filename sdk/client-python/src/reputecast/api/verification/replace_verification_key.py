from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.replace_verification_key_service import ReplaceVerificationKeyService
from ...models.verification_replace_key_input import VerificationReplaceKeyInput
from ...models.verification_service_view import VerificationServiceView
from typing import cast



def _get_kwargs(
    workspace: str,
    service: ReplaceVerificationKeyService,
    *,
    body: VerificationReplaceKeyInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/verification/services/{service}/key".format(workspace=quote(str(workspace), safe=""),service=quote(str(service), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VerificationServiceView:
    if response.status_code == 200:
        response_200 = VerificationServiceView.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VerificationServiceView]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    service: ReplaceVerificationKeyService,
    *,
    client: AuthenticatedClient,
    body: VerificationReplaceKeyInput,

) -> Response[Problem | VerificationServiceView]:
    """ Replace a service's API key (write-only)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (ReplaceVerificationKeyService):
        body (VerificationReplaceKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationServiceView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
service=service,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    service: ReplaceVerificationKeyService,
    *,
    client: AuthenticatedClient,
    body: VerificationReplaceKeyInput,

) -> Problem | VerificationServiceView | None:
    """ Replace a service's API key (write-only)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (ReplaceVerificationKeyService):
        body (VerificationReplaceKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationServiceView
     """


    return sync_detailed(
        workspace=workspace,
service=service,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    service: ReplaceVerificationKeyService,
    *,
    client: AuthenticatedClient,
    body: VerificationReplaceKeyInput,

) -> Response[Problem | VerificationServiceView]:
    """ Replace a service's API key (write-only)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (ReplaceVerificationKeyService):
        body (VerificationReplaceKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationServiceView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
service=service,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    service: ReplaceVerificationKeyService,
    *,
    client: AuthenticatedClient,
    body: VerificationReplaceKeyInput,

) -> Problem | VerificationServiceView | None:
    """ Replace a service's API key (write-only)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (ReplaceVerificationKeyService):
        body (VerificationReplaceKeyInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationServiceView
     """


    return (await asyncio_detailed(
        workspace=workspace,
service=service,
client=client,
body=body,

    )).parsed
