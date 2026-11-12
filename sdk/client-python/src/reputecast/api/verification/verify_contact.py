from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.contact_address_check import ContactAddressCheck
from ...models.problem import Problem
from ...models.verify_contact_input import VerifyContactInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact: UUID,
    *,
    body: VerifyContactInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/verification/contacts/{contact}/verify".format(workspace=quote(str(workspace), safe=""),contact=quote(str(contact), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ContactAddressCheck | Problem:
    if response.status_code == 200:
        response_200 = ContactAddressCheck.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ContactAddressCheck | Problem]:
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
    body: VerifyContactInput,

) -> Response[ContactAddressCheck | Problem]:
    """ Check one contact's address now (audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (VerifyContactInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactAddressCheck | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
body=body,

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
    body: VerifyContactInput,

) -> ContactAddressCheck | Problem | None:
    """ Check one contact's address now (audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (VerifyContactInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactAddressCheck | Problem
     """


    return sync_detailed(
        workspace=workspace,
contact=contact,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,
    body: VerifyContactInput,

) -> Response[ContactAddressCheck | Problem]:
    """ Check one contact's address now (audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (VerifyContactInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactAddressCheck | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
body=body,

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
    body: VerifyContactInput,

) -> ContactAddressCheck | Problem | None:
    """ Check one contact's address now (audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        body (VerifyContactInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactAddressCheck | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact=contact,
client=client,
body=body,

    )).parsed
