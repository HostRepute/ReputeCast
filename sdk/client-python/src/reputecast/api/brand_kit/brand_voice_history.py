from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.brand_voice_history import BrandVoiceHistory
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    before: int | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["before"] = before


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/brand-kit/voice/history".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BrandVoiceHistory | Problem:
    if response.status_code == 200:
        response_200 = BrandVoiceHistory.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BrandVoiceHistory | Problem]:
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
    before: int | Unset = UNSET,

) -> Response[BrandVoiceHistory | Problem]:
    """ Saved voice revisions, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BrandVoiceHistory | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
before=before,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    before: int | Unset = UNSET,

) -> BrandVoiceHistory | Problem | None:
    """ Saved voice revisions, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BrandVoiceHistory | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
before=before,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    before: int | Unset = UNSET,

) -> Response[BrandVoiceHistory | Problem]:
    """ Saved voice revisions, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BrandVoiceHistory | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
before=before,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    before: int | Unset = UNSET,

) -> BrandVoiceHistory | Problem | None:
    """ Saved voice revisions, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BrandVoiceHistory | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
before=before,

    )).parsed
