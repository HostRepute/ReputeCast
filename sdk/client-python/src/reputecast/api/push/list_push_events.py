from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.push_event_list import PushEventList
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    workspace: str,
    *,
    before: datetime.datetime | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_before: str | Unset = UNSET
    if not isinstance(before, Unset):
        json_before = before.isoformat()
    params["before"] = json_before

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/push/events".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | PushEventList:
    if response.status_code == 200:
        response_200 = PushEventList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | PushEventList]:
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
    before: datetime.datetime | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[Problem | PushEventList]:
    """ What the push services and the service workers reported

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (datetime.datetime | Unset): Events before this time
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushEventList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
before=before,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    before: datetime.datetime | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Problem | PushEventList | None:
    """ What the push services and the service workers reported

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (datetime.datetime | Unset): Events before this time
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PushEventList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
before=before,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    before: datetime.datetime | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[Problem | PushEventList]:
    """ What the push services and the service workers reported

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (datetime.datetime | Unset): Events before this time
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushEventList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
before=before,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    before: datetime.datetime | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Problem | PushEventList | None:
    """ What the push services and the service workers reported

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        before (datetime.datetime | Unset): Events before this time
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PushEventList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
before=before,
limit=limit,

    )).parsed
