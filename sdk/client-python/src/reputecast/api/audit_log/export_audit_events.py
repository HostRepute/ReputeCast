from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.export_audit_events_actor_type import ExportAuditEventsActorType
from ...models.export_audit_events_category import ExportAuditEventsCategory
from ...models.export_audit_events_format import ExportAuditEventsFormat
from ...models.export_audit_events_range import ExportAuditEventsRange
from ...models.export_audit_events_result import ExportAuditEventsResult
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,
    actor_type: ExportAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ExportAuditEventsCategory | Unset = UNSET,
    result: ExportAuditEventsResult | Unset = UNSET,
    range_: ExportAuditEventsRange | Unset = ExportAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    format_: ExportAuditEventsFormat | Unset = ExportAuditEventsFormat.CSV,

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

    json_format_: str | Unset = UNSET
    if not isinstance(format_, Unset):
        json_format_ = format_.value

    params["format"] = json_format_


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/audit/export".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Problem:
    if response.status_code == 200:
        response_200 = cast(Any, None)
        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Problem]:
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
    actor_type: ExportAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ExportAuditEventsCategory | Unset = UNSET,
    result: ExportAuditEventsResult | Unset = UNSET,
    range_: ExportAuditEventsRange | Unset = ExportAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    format_: ExportAuditEventsFormat | Unset = ExportAuditEventsFormat.CSV,

) -> Response[Any | Problem]:
    """ Export matching entries as CSV or JSON (streamed)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ExportAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ExportAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ExportAuditEventsResult | Unset):
        range_ (ExportAuditEventsRange | Unset):  Default: ExportAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        format_ (ExportAuditEventsFormat | Unset):  Default: ExportAuditEventsFormat.CSV.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
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
format_=format_,

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
    actor_type: ExportAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ExportAuditEventsCategory | Unset = UNSET,
    result: ExportAuditEventsResult | Unset = UNSET,
    range_: ExportAuditEventsRange | Unset = ExportAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    format_: ExportAuditEventsFormat | Unset = ExportAuditEventsFormat.CSV,

) -> Any | Problem | None:
    """ Export matching entries as CSV or JSON (streamed)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ExportAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ExportAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ExportAuditEventsResult | Unset):
        range_ (ExportAuditEventsRange | Unset):  Default: ExportAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        format_ (ExportAuditEventsFormat | Unset):  Default: ExportAuditEventsFormat.CSV.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
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
format_=format_,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    actor_type: ExportAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ExportAuditEventsCategory | Unset = UNSET,
    result: ExportAuditEventsResult | Unset = UNSET,
    range_: ExportAuditEventsRange | Unset = ExportAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    format_: ExportAuditEventsFormat | Unset = ExportAuditEventsFormat.CSV,

) -> Response[Any | Problem]:
    """ Export matching entries as CSV or JSON (streamed)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ExportAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ExportAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ExportAuditEventsResult | Unset):
        range_ (ExportAuditEventsRange | Unset):  Default: ExportAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        format_ (ExportAuditEventsFormat | Unset):  Default: ExportAuditEventsFormat.CSV.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
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
format_=format_,

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
    actor_type: ExportAuditEventsActorType | Unset = UNSET,
    actor: str | Unset = UNSET,
    action: str | Unset = UNSET,
    category: ExportAuditEventsCategory | Unset = UNSET,
    result: ExportAuditEventsResult | Unset = UNSET,
    range_: ExportAuditEventsRange | Unset = ExportAuditEventsRange.VALUE_1,
    since: datetime.datetime | Unset = UNSET,
    until: datetime.datetime | Unset = UNSET,
    format_: ExportAuditEventsFormat | Unset = ExportAuditEventsFormat.CSV,

) -> Any | Problem | None:
    """ Export matching entries as CSV or JSON (streamed)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Target name, target id, request id or actor name
        actor_type (ExportAuditEventsActorType | Unset):
        actor (str | Unset): Actor id
        action (str | Unset): Action group (routing) or action (routing.updated)
        category (ExportAuditEventsCategory | Unset): A group of actions, as in the action filter
        result (ExportAuditEventsResult | Unset):
        range_ (ExportAuditEventsRange | Unset):  Default: ExportAuditEventsRange.VALUE_1.
        since (datetime.datetime | Unset): Overrides range
        until (datetime.datetime | Unset):
        format_ (ExportAuditEventsFormat | Unset):  Default: ExportAuditEventsFormat.CSV.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
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
format_=format_,

    )).parsed
