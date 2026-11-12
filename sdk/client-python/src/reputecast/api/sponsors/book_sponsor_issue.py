from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.book_input import BookInput
from ...models.problem import Problem
from ...models.sponsor_booking_detail import SponsorBookingDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    number: int,
    *,
    body: BookInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/sponsor-slots/{id}/issues/{number}/book".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),number=quote(str(number), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SponsorBookingDetail:
    if response.status_code == 201:
        response_201 = SponsorBookingDetail.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SponsorBookingDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,
    body: BookInput,

) -> Response[Problem | SponsorBookingDetail]:
    """ Book an open issue for a sponsor, or hold it until a date

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number
        body (BookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorBookingDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
number=number,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,
    body: BookInput,

) -> Problem | SponsorBookingDetail | None:
    """ Book an open issue for a sponsor, or hold it until a date

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number
        body (BookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorBookingDetail
     """


    return sync_detailed(
        workspace=workspace,
id=id,
number=number,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,
    body: BookInput,

) -> Response[Problem | SponsorBookingDetail]:
    """ Book an open issue for a sponsor, or hold it until a date

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number
        body (BookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorBookingDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
number=number,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,
    body: BookInput,

) -> Problem | SponsorBookingDetail | None:
    """ Book an open issue for a sponsor, or hold it until a date

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number
        body (BookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorBookingDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
number=number,
client=client,
body=body,

    )).parsed
