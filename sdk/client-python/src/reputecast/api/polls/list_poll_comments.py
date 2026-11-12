from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_poll_comments_group import ListPollCommentsGroup
from ...models.poll_comment_list import PollCommentList
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    poll_id: UUID,
    *,
    group: ListPollCommentsGroup | Unset = ListPollCommentsGroup.ALL,
    search: str | Unset = UNSET,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_group: str | Unset = UNSET
    if not isinstance(group, Unset):
        json_group = group.value

    params["group"] = json_group

    params["search"] = search

    params["offset"] = offset

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/polls/{poll_id}/comments".format(workspace=quote(str(workspace), safe=""),poll_id=quote(str(poll_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PollCommentList | Problem:
    if response.status_code == 200:
        response_200 = PollCommentList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PollCommentList | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,
    group: ListPollCommentsGroup | Unset = ListPollCommentsGroup.ALL,
    search: str | Unset = UNSET,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Response[PollCommentList | Problem]:
    """ A poll's comments, newest first

     Contacts' names are included for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id
        group (ListPollCommentsGroup | Unset): Comments of 0–10 answers by group Default:
            ListPollCommentsGroup.ALL.
        search (str | Unset): Words in the comment (and the contact's name, with subscribers:view)
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PollCommentList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
poll_id=poll_id,
group=group,
search=search,
offset=offset,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,
    group: ListPollCommentsGroup | Unset = ListPollCommentsGroup.ALL,
    search: str | Unset = UNSET,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> PollCommentList | Problem | None:
    """ A poll's comments, newest first

     Contacts' names are included for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id
        group (ListPollCommentsGroup | Unset): Comments of 0–10 answers by group Default:
            ListPollCommentsGroup.ALL.
        search (str | Unset): Words in the comment (and the contact's name, with subscribers:view)
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PollCommentList | Problem
     """


    return sync_detailed(
        workspace=workspace,
poll_id=poll_id,
client=client,
group=group,
search=search,
offset=offset,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,
    group: ListPollCommentsGroup | Unset = ListPollCommentsGroup.ALL,
    search: str | Unset = UNSET,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Response[PollCommentList | Problem]:
    """ A poll's comments, newest first

     Contacts' names are included for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id
        group (ListPollCommentsGroup | Unset): Comments of 0–10 answers by group Default:
            ListPollCommentsGroup.ALL.
        search (str | Unset): Words in the comment (and the contact's name, with subscribers:view)
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PollCommentList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
poll_id=poll_id,
group=group,
search=search,
offset=offset,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    poll_id: UUID,
    *,
    client: AuthenticatedClient,
    group: ListPollCommentsGroup | Unset = ListPollCommentsGroup.ALL,
    search: str | Unset = UNSET,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> PollCommentList | Problem | None:
    """ A poll's comments, newest first

     Contacts' names are included for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        poll_id (UUID): Poll id
        group (ListPollCommentsGroup | Unset): Comments of 0–10 answers by group Default:
            ListPollCommentsGroup.ALL.
        search (str | Unset): Words in the comment (and the contact's name, with subscribers:view)
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PollCommentList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
poll_id=poll_id,
client=client,
group=group,
search=search,
offset=offset,
limit=limit,

    )).parsed
