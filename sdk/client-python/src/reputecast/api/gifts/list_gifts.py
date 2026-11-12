from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.gift_list import GiftList
from ...models.list_gifts_status import ListGiftsStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    status: ListGiftsStatus | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["q"] = q

    params["page"] = page


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/earn/gifts/list".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> GiftList | Problem:
    if response.status_code == 200:
        response_200 = GiftList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[GiftList | Problem]:
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
    status: ListGiftsStatus | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> Response[GiftList | Problem]:
    """ A page of the gifts bought on the gift page, the latest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListGiftsStatus | Unset): The filter (all by default)
        q (str | Unset): Buyer, recipient or email
        page (int | Unset): The page (20 gifts each)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GiftList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
q=q,
page=page,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListGiftsStatus | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> GiftList | Problem | None:
    """ A page of the gifts bought on the gift page, the latest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListGiftsStatus | Unset): The filter (all by default)
        q (str | Unset): Buyer, recipient or email
        page (int | Unset): The page (20 gifts each)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GiftList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,
q=q,
page=page,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListGiftsStatus | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> Response[GiftList | Problem]:
    """ A page of the gifts bought on the gift page, the latest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListGiftsStatus | Unset): The filter (all by default)
        q (str | Unset): Buyer, recipient or email
        page (int | Unset): The page (20 gifts each)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GiftList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
q=q,
page=page,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListGiftsStatus | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = UNSET,

) -> GiftList | Problem | None:
    """ A page of the gifts bought on the gift page, the latest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListGiftsStatus | Unset): The filter (all by default)
        q (str | Unset): Buyer, recipient or email
        page (int | Unset): The page (20 gifts each)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GiftList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,
q=q,
page=page,

    )).parsed
