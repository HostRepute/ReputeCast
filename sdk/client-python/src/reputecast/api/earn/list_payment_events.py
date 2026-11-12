from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.earn_event_log import EarnEventLog
from ...models.list_payment_events_group import ListPaymentEventsGroup
from ...models.list_payment_events_range import ListPaymentEventsRange
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    group: ListPaymentEventsGroup | Unset = ListPaymentEventsGroup.ALL,
    range_: ListPaymentEventsRange | Unset = ListPaymentEventsRange.VALUE_1,
    q: str | Unset = UNSET,
    test: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_group: str | Unset = UNSET
    if not isinstance(group, Unset):
        json_group = group.value

    params["group"] = json_group

    json_range_: str | Unset = UNSET
    if not isinstance(range_, Unset):
        json_range_ = range_.value

    params["range"] = json_range_

    params["q"] = q

    params["test"] = test

    params["page"] = page

    params["pageSize"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/earn/payments/events".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EarnEventLog | Problem:
    if response.status_code == 200:
        response_200 = EarnEventLog.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EarnEventLog | Problem]:
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
    group: ListPaymentEventsGroup | Unset = ListPaymentEventsGroup.ALL,
    range_: ListPaymentEventsRange | Unset = ListPaymentEventsRange.VALUE_1,
    q: str | Unset = UNSET,
    test: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Response[EarnEventLog | Problem]:
    """ The payments event log: each provider event and what ReputeCast did with it

     Raw events are kept 90 days. Payment events never go to the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        group (ListPaymentEventsGroup | Unset): A group of event types Default:
            ListPaymentEventsGroup.ALL.
        range_ (ListPaymentEventsRange | Unset): How far back (raw events are kept 90 days)
            Default: ListPaymentEventsRange.VALUE_1.
        q (str | Unset): Search the event type, the event id, the paid member or the raw event
        test (bool | Unset): Include test-mode events
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EarnEventLog | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
group=group,
range_=range_,
q=q,
test=test,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    group: ListPaymentEventsGroup | Unset = ListPaymentEventsGroup.ALL,
    range_: ListPaymentEventsRange | Unset = ListPaymentEventsRange.VALUE_1,
    q: str | Unset = UNSET,
    test: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> EarnEventLog | Problem | None:
    """ The payments event log: each provider event and what ReputeCast did with it

     Raw events are kept 90 days. Payment events never go to the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        group (ListPaymentEventsGroup | Unset): A group of event types Default:
            ListPaymentEventsGroup.ALL.
        range_ (ListPaymentEventsRange | Unset): How far back (raw events are kept 90 days)
            Default: ListPaymentEventsRange.VALUE_1.
        q (str | Unset): Search the event type, the event id, the paid member or the raw event
        test (bool | Unset): Include test-mode events
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EarnEventLog | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
group=group,
range_=range_,
q=q,
test=test,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    group: ListPaymentEventsGroup | Unset = ListPaymentEventsGroup.ALL,
    range_: ListPaymentEventsRange | Unset = ListPaymentEventsRange.VALUE_1,
    q: str | Unset = UNSET,
    test: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> Response[EarnEventLog | Problem]:
    """ The payments event log: each provider event and what ReputeCast did with it

     Raw events are kept 90 days. Payment events never go to the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        group (ListPaymentEventsGroup | Unset): A group of event types Default:
            ListPaymentEventsGroup.ALL.
        range_ (ListPaymentEventsRange | Unset): How far back (raw events are kept 90 days)
            Default: ListPaymentEventsRange.VALUE_1.
        q (str | Unset): Search the event type, the event id, the paid member or the raw event
        test (bool | Unset): Include test-mode events
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EarnEventLog | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
group=group,
range_=range_,
q=q,
test=test,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    group: ListPaymentEventsGroup | Unset = ListPaymentEventsGroup.ALL,
    range_: ListPaymentEventsRange | Unset = ListPaymentEventsRange.VALUE_1,
    q: str | Unset = UNSET,
    test: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 8,

) -> EarnEventLog | Problem | None:
    """ The payments event log: each provider event and what ReputeCast did with it

     Raw events are kept 90 days. Payment events never go to the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        group (ListPaymentEventsGroup | Unset): A group of event types Default:
            ListPaymentEventsGroup.ALL.
        range_ (ListPaymentEventsRange | Unset): How far back (raw events are kept 90 days)
            Default: ListPaymentEventsRange.VALUE_1.
        q (str | Unset): Search the event type, the event id, the paid member or the raw event
        test (bool | Unset): Include test-mode events
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EarnEventLog | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
group=group,
range_=range_,
q=q,
test=test,
page=page,
page_size=page_size,

    )).parsed
