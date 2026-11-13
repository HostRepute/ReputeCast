from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_replies_view import ListRepliesView
from ...models.problem import Problem
from ...models.reply_list import ReplyList
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    *,
    view: ListRepliesView | Unset = ListRepliesView.OPEN,
    campaign: UUID | Unset = UNSET,
    automation: UUID | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_view: str | Unset = UNSET
    if not isinstance(view, Unset):
        json_view = view.value

    params["view"] = json_view

    json_campaign: str | Unset = UNSET
    if not isinstance(campaign, Unset):
        json_campaign = str(campaign)
    params["campaign"] = json_campaign

    json_automation: str | Unset = UNSET
    if not isinstance(automation, Unset):
        json_automation = str(automation)
    params["automation"] = json_automation

    params["q"] = q

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/replies".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReplyList:
    if response.status_code == 200:
        response_200 = ReplyList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReplyList]:
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
    view: ListRepliesView | Unset = ListRepliesView.OPEN,
    campaign: UUID | Unset = UNSET,
    automation: UUID | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[Problem | ReplyList]:
    """ List replies and other received mail

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        view (ListRepliesView | Unset):  Default: ListRepliesView.OPEN.
        campaign (UUID | Unset): Only replies to this campaign
        automation (UUID | Unset): Only replies to this flow's emails
        q (str | Unset): Search the sender and subject
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReplyList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
view=view,
campaign=campaign,
automation=automation,
q=q,
cursor=cursor,
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
    view: ListRepliesView | Unset = ListRepliesView.OPEN,
    campaign: UUID | Unset = UNSET,
    automation: UUID | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Problem | ReplyList | None:
    """ List replies and other received mail

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        view (ListRepliesView | Unset):  Default: ListRepliesView.OPEN.
        campaign (UUID | Unset): Only replies to this campaign
        automation (UUID | Unset): Only replies to this flow's emails
        q (str | Unset): Search the sender and subject
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReplyList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
view=view,
campaign=campaign,
automation=automation,
q=q,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    view: ListRepliesView | Unset = ListRepliesView.OPEN,
    campaign: UUID | Unset = UNSET,
    automation: UUID | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[Problem | ReplyList]:
    """ List replies and other received mail

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        view (ListRepliesView | Unset):  Default: ListRepliesView.OPEN.
        campaign (UUID | Unset): Only replies to this campaign
        automation (UUID | Unset): Only replies to this flow's emails
        q (str | Unset): Search the sender and subject
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReplyList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
view=view,
campaign=campaign,
automation=automation,
q=q,
cursor=cursor,
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
    view: ListRepliesView | Unset = ListRepliesView.OPEN,
    campaign: UUID | Unset = UNSET,
    automation: UUID | Unset = UNSET,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Problem | ReplyList | None:
    """ List replies and other received mail

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        view (ListRepliesView | Unset):  Default: ListRepliesView.OPEN.
        campaign (UUID | Unset): Only replies to this campaign
        automation (UUID | Unset): Only replies to this flow's emails
        q (str | Unset): Search the sender and subject
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReplyList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
view=view,
campaign=campaign,
automation=automation,
q=q,
cursor=cursor,
limit=limit,

    )).parsed
