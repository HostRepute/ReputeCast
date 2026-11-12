from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.earn_audience import EarnAudience
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    *,
    list_: UUID | Unset = UNSET,
    tier: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_list_: str | Unset = UNSET
    if not isinstance(list_, Unset):
        json_list_ = str(list_)
    params["list"] = json_list_

    params["tier"] = tier


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/earn/audience".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EarnAudience | Problem:
    if response.status_code == 200:
        response_200 = EarnAudience.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EarnAudience | Problem]:
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
    list_: UUID | Unset = UNSET,
    tier: str | Unset = UNSET,

) -> Response[EarnAudience | Problem]:
    """ Who on a list a paid-only campaign reaches: everyone, paid members only or free subscribers only

     Counts only. Drafting a paid-only campaign needs campaigns:edit, like any draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID | Unset): The campaign's list
        tier (str | Unset): The lowest tier of paid members; empty for any paid tier

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EarnAudience | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
list_=list_,
tier=tier,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    list_: UUID | Unset = UNSET,
    tier: str | Unset = UNSET,

) -> EarnAudience | Problem | None:
    """ Who on a list a paid-only campaign reaches: everyone, paid members only or free subscribers only

     Counts only. Drafting a paid-only campaign needs campaigns:edit, like any draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID | Unset): The campaign's list
        tier (str | Unset): The lowest tier of paid members; empty for any paid tier

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EarnAudience | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
list_=list_,
tier=tier,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    list_: UUID | Unset = UNSET,
    tier: str | Unset = UNSET,

) -> Response[EarnAudience | Problem]:
    """ Who on a list a paid-only campaign reaches: everyone, paid members only or free subscribers only

     Counts only. Drafting a paid-only campaign needs campaigns:edit, like any draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID | Unset): The campaign's list
        tier (str | Unset): The lowest tier of paid members; empty for any paid tier

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EarnAudience | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
list_=list_,
tier=tier,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    list_: UUID | Unset = UNSET,
    tier: str | Unset = UNSET,

) -> EarnAudience | Problem | None:
    """ Who on a list a paid-only campaign reaches: everyone, paid members only or free subscribers only

     Counts only. Drafting a paid-only campaign needs campaigns:edit, like any draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID | Unset): The campaign's list
        tier (str | Unset): The lowest tier of paid members; empty for any paid tier

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EarnAudience | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
list_=list_,
tier=tier,

    )).parsed
