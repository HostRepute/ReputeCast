from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_verification_realtime_service import GetVerificationRealtimeService
from ...models.problem import Problem
from ...models.verification_realtime import VerificationRealtime
from typing import cast



def _get_kwargs(
    workspace: str,
    service: GetVerificationRealtimeService,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/verification/services/{service}/realtime".format(workspace=quote(str(workspace), safe=""),service=quote(str(service), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VerificationRealtime:
    if response.status_code == 200:
        response_200 = VerificationRealtime.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VerificationRealtime]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    service: GetVerificationRealtimeService,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | VerificationRealtime]:
    """ A service's real-time checks by form and by day

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (GetVerificationRealtimeService):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationRealtime]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
service=service,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    service: GetVerificationRealtimeService,
    *,
    client: AuthenticatedClient,

) -> Problem | VerificationRealtime | None:
    """ A service's real-time checks by form and by day

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (GetVerificationRealtimeService):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationRealtime
     """


    return sync_detailed(
        workspace=workspace,
service=service,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    service: GetVerificationRealtimeService,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | VerificationRealtime]:
    """ A service's real-time checks by form and by day

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (GetVerificationRealtimeService):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationRealtime]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
service=service,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    service: GetVerificationRealtimeService,
    *,
    client: AuthenticatedClient,

) -> Problem | VerificationRealtime | None:
    """ A service's real-time checks by form and by day

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        service (GetVerificationRealtimeService):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationRealtime
     """


    return (await asyncio_detailed(
        workspace=workspace,
service=service,
client=client,

    )).parsed
