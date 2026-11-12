from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.audit_event_list import AuditEventList
from ...models.list_audit_events_actor_type import ListAuditEventsActorType
from ...models.list_audit_events_category import ListAuditEventsCategory
from ...models.list_audit_events_range import ListAuditEventsRange
from ...models.list_audit_events_result import ListAuditEventsResult
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,
    actor_type: ListAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ListAuditEventsCategory | Unset = UNSET,
    result: ListAuditEventsResult | Unset = UNSET,
    range_: ListAuditEventsRange | Unset = ListAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q

    json_actor_type: str | Unset = UNSET
    if not isinstance(actor_type, Unset):
        json_actor_type = actor_type.value

    params["actorType"] = json_actor_type

    params["actor"] = actor

    params["action"] = action

    json_category: str | Unset = UNSET
    if not isinstance(category, Unset):
        json_category = category.value

    params["category"] = json_category

    json_result: str | Unset = UNSET
    if not isinstance(result, Unset):
        json_result = result.value

    params["result"] = json_result

    json_range_: str | Unset = UNSET
    if not isinstance(range_, Unset):
        json_range_ = range_.value

    params["range"] = json_range_

    json_since: str | Unset = UNSET
    if not isinstance(since, Unset):
        json_since = since.isoformat()
    params["since"] = json_since

    json_until: str | Unset = UNSET
    if not isinstance(until, Unset):
        json_until = until.isoformat()
    params["until"] = json_until

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/audit".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AuditEventList | Problem:
    if response.status_code == 200:
        response_200 = AuditEventList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AuditEventList | Problem]:
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
    q: str | Unset = UNSET,
    actor_type: ListAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ListAuditEventsCategory | Unset = UNSET,
    result: ListAuditEventsResult | Unset = UNSET,
    range_: ListAuditEventsRange | Unset = ListAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[AuditEventList | Problem]:
    """ List audit entries with filters, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ListAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ListAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ListAuditEventsResult | Unset):
        range_ (ListAuditEventsRange | Unset):  Default: ListAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AuditEventList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
actor_type=actor_type,
actor=actor,
action=action,
category=category,
result=result,
range_=range_,
since=since,
until=until,
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
    q: str | Unset = UNSET,
    actor_type: ListAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ListAuditEventsCategory | Unset = UNSET,
    result: ListAuditEventsResult | Unset = UNSET,
    range_: ListAuditEventsRange | Unset = ListAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> AuditEventList | Problem | None:
    """ List audit entries with filters, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ListAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ListAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ListAuditEventsResult | Unset):
        range_ (ListAuditEventsRange | Unset):  Default: ListAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AuditEventList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,
actor_type=actor_type,
actor=actor,
action=action,
category=category,
result=result,
range_=range_,
since=since,
until=until,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    actor_type: ListAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ListAuditEventsCategory | Unset = UNSET,
    result: ListAuditEventsResult | Unset = UNSET,
    range_: ListAuditEventsRange | Unset = ListAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[AuditEventList | Problem]:
    """ List audit entries with filters, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ListAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ListAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ListAuditEventsResult | Unset):
        range_ (ListAuditEventsRange | Unset):  Default: ListAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AuditEventList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
actor_type=actor_type,
actor=actor,
action=action,
category=category,
result=result,
range_=range_,
since=since,
until=until,
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
    q: str | Unset = UNSET,
    actor_type: ListAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ListAuditEventsCategory | Unset = UNSET,
    result: ListAuditEventsResult | Unset = UNSET,
    range_: ListAuditEventsRange | Unset = ListAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> AuditEventList | Problem | None:
    """ List audit entries with filters, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ListAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ListAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ListAuditEventsResult | Unset):
        range_ (ListAuditEventsRange | Unset):  Default: ListAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AuditEventList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,
actor_type=actor_type,
actor=actor,
action=action,
category=category,
result=result,
range_=range_,
since=since,
until=until,
cursor=cursor,
limit=limit,

    )).parsed
