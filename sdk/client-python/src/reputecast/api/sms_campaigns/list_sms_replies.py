from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_sms_replies_kind import ListSmsRepliesKind
from ...models.problem import Problem
from ...models.sms_reply_list import SmsReplyList
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    kind: ListSmsRepliesKind | Unset = ListSmsRepliesKind.ALL,
    offset: int | Unset = 0,
    limit: int | Unset = 50,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_kind: str | Unset = UNSET
    if not isinstance(kind, Unset):
        json_kind = kind.value

    params["kind"] = json_kind

    params["offset"] = offset

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/sms-campaigns/{campaign_id}/replies".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsReplyList:
    if response.status_code == 200:
        response_200 = SmsReplyList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsReplyList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    kind: ListSmsRepliesKind | Unset = ListSmsRepliesKind.ALL,
    offset: int | Unset = 0,
    limit: int | Unset = 50,

) -> Response[Problem | SmsReplyList]:
    """ A campaign's replies: STOP, HELP and others

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        kind (ListSmsRepliesKind | Unset):  Default: ListSmsRepliesKind.ALL.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsReplyList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
kind=kind,
offset=offset,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    kind: ListSmsRepliesKind | Unset = ListSmsRepliesKind.ALL,
    offset: int | Unset = 0,
    limit: int | Unset = 50,

) -> Problem | SmsReplyList | None:
    """ A campaign's replies: STOP, HELP and others

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        kind (ListSmsRepliesKind | Unset):  Default: ListSmsRepliesKind.ALL.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsReplyList
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
kind=kind,
offset=offset,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    kind: ListSmsRepliesKind | Unset = ListSmsRepliesKind.ALL,
    offset: int | Unset = 0,
    limit: int | Unset = 50,

) -> Response[Problem | SmsReplyList]:
    """ A campaign's replies: STOP, HELP and others

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        kind (ListSmsRepliesKind | Unset):  Default: ListSmsRepliesKind.ALL.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsReplyList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
kind=kind,
offset=offset,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    kind: ListSmsRepliesKind | Unset = ListSmsRepliesKind.ALL,
    offset: int | Unset = 0,
    limit: int | Unset = 50,

) -> Problem | SmsReplyList | None:
    """ A campaign's replies: STOP, HELP and others

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        kind (ListSmsRepliesKind | Unset):  Default: ListSmsRepliesKind.ALL.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsReplyList
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
kind=kind,
offset=offset,
limit=limit,

    )).parsed
