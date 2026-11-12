from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.update_verification_service_service import UpdateVerificationServiceService
from ...models.verification_service_view import VerificationServiceView
from ...models.verification_settings_input import VerificationSettingsInput
from typing import cast



def _get_kwargs(
    workspace: str,
    service: UpdateVerificationServiceService,
    *,
    body: VerificationSettingsInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/workspaces/{workspace}/verification/services/{service}".format(workspace=quote(str(workspace), safe=""),service=quote(str(service), safe=""),),
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
    service: UpdateVerificationServiceService,
    *,
    client: AuthenticatedClient,
    body: VerificationSettingsInput,

) -> Response[Problem | VerificationServiceView]:
    """ Change a service's price per check, region and default uses

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (UpdateVerificationServiceService):
        body (VerificationSettingsInput):

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
    service: UpdateVerificationServiceService,
    *,
    client: AuthenticatedClient,
    body: VerificationSettingsInput,

) -> Problem | VerificationServiceView | None:
    """ Change a service's price per check, region and default uses

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (UpdateVerificationServiceService):
        body (VerificationSettingsInput):

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
    service: UpdateVerificationServiceService,
    *,
    client: AuthenticatedClient,
    body: VerificationSettingsInput,

) -> Response[Problem | VerificationServiceView]:
    """ Change a service's price per check, region and default uses

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (UpdateVerificationServiceService):
        body (VerificationSettingsInput):

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
    service: UpdateVerificationServiceService,
    *,
    client: AuthenticatedClient,
    body: VerificationSettingsInput,

) -> Problem | VerificationServiceView | None:
    """ Change a service's price per check, region and default uses

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (UpdateVerificationServiceService):
        body (VerificationSettingsInput):

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
