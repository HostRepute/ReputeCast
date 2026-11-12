from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.audit_event_detail import AuditEventDetail
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    event_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/audit/events/{event_id}".format(workspace=quote(str(workspace), safe=""),event_id=quote(str(event_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AuditEventDetail | Problem:
    if response.status_code == 200:
        response_200 = AuditEventDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AuditEventDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    event_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[AuditEventDetail | Problem]:
    """ One audit entry with before/after values, IP, user agent and request id

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        event_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AuditEventDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
event_id=event_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    event_id: UUID,
    *,
    client: AuthenticatedClient,

) -> AuditEventDetail | Problem | None:
    """ One audit entry with before/after values, IP, user agent and request id

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        event_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AuditEventDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
event_id=event_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    event_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[AuditEventDetail | Problem]:
    """ One audit entry with before/after values, IP, user agent and request id

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        event_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AuditEventDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
event_id=event_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    event_id: UUID,
    *,
    client: AuthenticatedClient,

) -> AuditEventDetail | Problem | None:
    """ One audit entry with before/after values, IP, user agent and request id

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        event_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AuditEventDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
event_id=event_id,
client=client,

    )).parsed
