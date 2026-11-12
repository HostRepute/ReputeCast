from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.disconnect_verification_service_service import DisconnectVerificationServiceService
from ...models.problem import Problem
from ...models.verification_disconnect_result import VerificationDisconnectResult
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    service: DisconnectVerificationServiceService,
    *,
    forms: list[str] | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_forms: list[str] | Unset = UNSET
    if not isinstance(forms, Unset):
        json_forms = forms


    params["forms"] = json_forms


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/workspaces/{workspace}/verification/services/{service}".format(workspace=quote(str(workspace), safe=""),service=quote(str(service), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VerificationDisconnectResult:
    if response.status_code == 200:
        response_200 = VerificationDisconnectResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VerificationDisconnectResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    service: DisconnectVerificationServiceService,
    *,
    client: AuthenticatedClient,
    forms: list[str] | Unset = UNSET,

) -> Response[Problem | VerificationDisconnectResult]:
    """ Disconnect a service; results already on contacts stay

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (DisconnectVerificationServiceService):
        forms (list[str] | Unset): The forms that stop checking, as the dialog listed them

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationDisconnectResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
service=service,
forms=forms,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    service: DisconnectVerificationServiceService,
    *,
    client: AuthenticatedClient,
    forms: list[str] | Unset = UNSET,

) -> Problem | VerificationDisconnectResult | None:
    """ Disconnect a service; results already on contacts stay

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (DisconnectVerificationServiceService):
        forms (list[str] | Unset): The forms that stop checking, as the dialog listed them

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationDisconnectResult
     """


    return sync_detailed(
        workspace=workspace,
service=service,
client=client,
forms=forms,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    service: DisconnectVerificationServiceService,
    *,
    client: AuthenticatedClient,
    forms: list[str] | Unset = UNSET,

) -> Response[Problem | VerificationDisconnectResult]:
    """ Disconnect a service; results already on contacts stay

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (DisconnectVerificationServiceService):
        forms (list[str] | Unset): The forms that stop checking, as the dialog listed them

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationDisconnectResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
service=service,
forms=forms,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    service: DisconnectVerificationServiceService,
    *,
    client: AuthenticatedClient,
    forms: list[str] | Unset = UNSET,

) -> Problem | VerificationDisconnectResult | None:
    """ Disconnect a service; results already on contacts stay

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (DisconnectVerificationServiceService):
        forms (list[str] | Unset): The forms that stop checking, as the dialog listed them

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationDisconnectResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
service=service,
client=client,
forms=forms,

    )).parsed
