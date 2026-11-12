from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.whats_app_consent_input import WhatsAppConsentInput
from ...models.whats_app_contact_card import WhatsAppContactCard
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact_id: UUID,
    *,
    body: WhatsAppConsentInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/contacts/{contact_id}/whatsapp/consent".format(workspace=quote(str(workspace), safe=""),contact_id=quote(str(contact_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WhatsAppContactCard:
    if response.status_code == 200:
        response_200 = WhatsAppContactCard.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WhatsAppContactCard]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WhatsAppConsentInput,

) -> Response[Problem | WhatsAppContactCard]:
    """ Change a contact's WhatsApp consent on a list (logged)

     Consent needs its evidence; an opt-out by the person can't be overridden.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID):
        body (WhatsAppConsentInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppContactCard]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact_id=contact_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WhatsAppConsentInput,

) -> Problem | WhatsAppContactCard | None:
    """ Change a contact's WhatsApp consent on a list (logged)

     Consent needs its evidence; an opt-out by the person can't be overridden.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID):
        body (WhatsAppConsentInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppContactCard
     """


    return sync_detailed(
        workspace=workspace,
contact_id=contact_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WhatsAppConsentInput,

) -> Response[Problem | WhatsAppContactCard]:
    """ Change a contact's WhatsApp consent on a list (logged)

     Consent needs its evidence; an opt-out by the person can't be overridden.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID):
        body (WhatsAppConsentInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppContactCard]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact_id=contact_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    contact_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WhatsAppConsentInput,

) -> Problem | WhatsAppContactCard | None:
    """ Change a contact's WhatsApp consent on a list (logged)

     Consent needs its evidence; an opt-out by the person can't be overridden.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (UUID):
        body (WhatsAppConsentInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppContactCard
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact_id=contact_id,
client=client,
body=body,

    )).parsed
