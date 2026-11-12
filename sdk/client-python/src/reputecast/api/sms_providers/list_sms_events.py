from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_sms_events_kind import ListSmsEventsKind
from ...models.problem import Problem
from ...models.sms_event_list import SmsEventList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    kind: ListSmsEventsKind | Unset = ListSmsEventsKind.ALL,
    provider: str | Unset = UNSET,
    page: int | Unset = 1,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_kind: str | Unset = UNSET
    if not isinstance(kind, Unset):
        json_kind = kind.value

    params["kind"] = json_kind

    params["provider"] = provider

    params["page"] = page


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/sms/events".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsEventList:
    if response.status_code == 200:
        response_200 = SmsEventList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsEventList]:
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
    kind: ListSmsEventsKind | Unset = ListSmsEventsKind.ALL,
    provider: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Response[Problem | SmsEventList]:
    """ Raw provider events beside their normalized ones, and opt-out sync

     Raw events are kept 90 days.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (ListSmsEventsKind | Unset):  Default: ListSmsEventsKind.ALL.
        provider (str | Unset): An SMS provider id
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsEventList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,
provider=provider,
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
    kind: ListSmsEventsKind | Unset = ListSmsEventsKind.ALL,
    provider: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Problem | SmsEventList | None:
    """ Raw provider events beside their normalized ones, and opt-out sync

     Raw events are kept 90 days.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (ListSmsEventsKind | Unset):  Default: ListSmsEventsKind.ALL.
        provider (str | Unset): An SMS provider id
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsEventList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
kind=kind,
provider=provider,
page=page,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    kind: ListSmsEventsKind | Unset = ListSmsEventsKind.ALL,
    provider: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Response[Problem | SmsEventList]:
    """ Raw provider events beside their normalized ones, and opt-out sync

     Raw events are kept 90 days.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (ListSmsEventsKind | Unset):  Default: ListSmsEventsKind.ALL.
        provider (str | Unset): An SMS provider id
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsEventList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,
provider=provider,
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
    kind: ListSmsEventsKind | Unset = ListSmsEventsKind.ALL,
    provider: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Problem | SmsEventList | None:
    """ Raw provider events beside their normalized ones, and opt-out sync

     Raw events are kept 90 days.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (ListSmsEventsKind | Unset):  Default: ListSmsEventsKind.ALL.
        provider (str | Unset): An SMS provider id
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsEventList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
kind=kind,
provider=provider,
page=page,

    )).parsed
