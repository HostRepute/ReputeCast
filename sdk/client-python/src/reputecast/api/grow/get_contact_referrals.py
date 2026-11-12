from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.referral_contact import ReferralContact
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/contacts/{contact}/referrals".format(workspace=quote(str(workspace), safe=""),contact=quote(str(contact), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReferralContact:
    if response.status_code == 200:
        response_200 = ReferralContact.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReferralContact]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralContact]:
    """ A contact's referral link and counts, who referred them, and Grow's timeline events

     API keys with grow:view can read a contact's referral link and counts, so an app can show them to
    its users.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralContact]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralContact | None:
    """ A contact's referral link and counts, who referred them, and Grow's timeline events

     API keys with grow:view can read a contact's referral link and counts, so an app can show them to
    its users.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralContact
     """


    return sync_detailed(
        workspace=workspace,
contact=contact,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralContact]:
    """ A contact's referral link and counts, who referred them, and Grow's timeline events

     API keys with grow:view can read a contact's referral link and counts, so an app can show them to
    its users.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralContact]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralContact | None:
    """ A contact's referral link and counts, who referred them, and Grow's timeline events

     API keys with grow:view can read a contact's referral link and counts, so an app can show them to
    its users.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralContact
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact=contact,
client=client,

    )).parsed
