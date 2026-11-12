from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.stream_live_send_event_heartbeat import StreamLiveSendEventHeartbeat
from ...models.stream_live_send_event_snapshot import StreamLiveSendEventSnapshot
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    last_event_id_query: str | Unset = UNSET,
    lang: str | Unset = UNSET,
    last_event_id_header: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(last_event_id_header, Unset):
        headers["Last-Event-ID"] = last_event_id_header



    

    params: dict[str, Any] = {}

    params["lastEventId"] = last_event_id_query

    params["lang"] = lang


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/live/stream".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
        "params": params,
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]:
    if response.status_code == 200:
        response_200 = []
        _response_200 = response.text
        for response_200_item_data in (_response_200):
            def _parse_response_200_item(data: object) -> StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot:
                try:
                    if not isinstance(data, dict):
                        raise TypeError()
                    response_200_item_event_heartbeat = StreamLiveSendEventHeartbeat.from_dict(data)



                    return response_200_item_event_heartbeat
                except (TypeError, ValueError, AttributeError, KeyError):
                    pass
                if not isinstance(data, dict):
                    raise TypeError()
                response_200_item_event_snapshot = StreamLiveSendEventSnapshot.from_dict(data)



                return response_200_item_event_snapshot

            response_200_item = _parse_response_200_item(response_200_item_data)

            response_200.append(response_200_item)

        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]]:
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
    last_event_id_query: str | Unset = UNSET,
    lang: str | Unset = UNSET,
    last_event_id_header: str | Unset = UNSET,

) -> Response[Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]]:
    """ Server-sent events of a send: a snapshot event at once and on every change, a heartbeat event every
    15 s

     Each `snapshot` event carries the whole live state; its id is the snapshot version, so a
    reconnecting EventSource resumes with Last-Event-ID. A `heartbeat` event every 15 s says the
    connection is alive while nothing changes: a monitor that hears nothing for longer treats its data
    as stale.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        last_event_id_query (str | Unset): Same as the Last-Event-ID header, for clients that
            reconnect by themselves
        lang (str | Unset): The language the snapshots' words read in (en, vi); like Accept-
            Language, for clients that can't set headers
        last_event_id_header (str | Unset): Resume after this event id (the snapshot version)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
last_event_id_query=last_event_id_query,
lang=lang,
last_event_id_header=last_event_id_header,

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
    last_event_id_query: str | Unset = UNSET,
    lang: str | Unset = UNSET,
    last_event_id_header: str | Unset = UNSET,

) -> Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot] | None:
    """ Server-sent events of a send: a snapshot event at once and on every change, a heartbeat event every
    15 s

     Each `snapshot` event carries the whole live state; its id is the snapshot version, so a
    reconnecting EventSource resumes with Last-Event-ID. A `heartbeat` event every 15 s says the
    connection is alive while nothing changes: a monitor that hears nothing for longer treats its data
    as stale.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        last_event_id_query (str | Unset): Same as the Last-Event-ID header, for clients that
            reconnect by themselves
        lang (str | Unset): The language the snapshots' words read in (en, vi); like Accept-
            Language, for clients that can't set headers
        last_event_id_header (str | Unset): Resume after this event id (the snapshot version)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
last_event_id_query=last_event_id_query,
lang=lang,
last_event_id_header=last_event_id_header,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    last_event_id_query: str | Unset = UNSET,
    lang: str | Unset = UNSET,
    last_event_id_header: str | Unset = UNSET,

) -> Response[Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]]:
    """ Server-sent events of a send: a snapshot event at once and on every change, a heartbeat event every
    15 s

     Each `snapshot` event carries the whole live state; its id is the snapshot version, so a
    reconnecting EventSource resumes with Last-Event-ID. A `heartbeat` event every 15 s says the
    connection is alive while nothing changes: a monitor that hears nothing for longer treats its data
    as stale.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        last_event_id_query (str | Unset): Same as the Last-Event-ID header, for clients that
            reconnect by themselves
        lang (str | Unset): The language the snapshots' words read in (en, vi); like Accept-
            Language, for clients that can't set headers
        last_event_id_header (str | Unset): Resume after this event id (the snapshot version)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
last_event_id_query=last_event_id_query,
lang=lang,
last_event_id_header=last_event_id_header,

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
    last_event_id_query: str | Unset = UNSET,
    lang: str | Unset = UNSET,
    last_event_id_header: str | Unset = UNSET,

) -> Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot] | None:
    """ Server-sent events of a send: a snapshot event at once and on every change, a heartbeat event every
    15 s

     Each `snapshot` event carries the whole live state; its id is the snapshot version, so a
    reconnecting EventSource resumes with Last-Event-ID. A `heartbeat` event every 15 s says the
    connection is alive while nothing changes: a monitor that hears nothing for longer treats its data
    as stale.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        last_event_id_query (str | Unset): Same as the Last-Event-ID header, for clients that
            reconnect by themselves
        lang (str | Unset): The language the snapshots' words read in (en, vi); like Accept-
            Language, for clients that can't set headers
        last_event_id_header (str | Unset): Resume after this event id (the snapshot version)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | list[StreamLiveSendEventHeartbeat | StreamLiveSendEventSnapshot]
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
last_event_id_query=last_event_id_query,
lang=lang,
last_event_id_header=last_event_id_header,

    )).parsed
