from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.campaign_list import CampaignList
from ...models.list_campaigns_channel import ListCampaignsChannel
from ...models.list_campaigns_status import ListCampaignsStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    status: ListCampaignsStatus | Unset = ListCampaignsStatus.ALL,
    q: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    sender: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    channel: ListCampaignsChannel | Unset = UNSET,
    paid_only: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["q"] = q

    params["from"] = from_

    params["to"] = to

    params["sender"] = sender

    params["list"] = list_

    params["tag"] = tag

    json_channel: str | Unset = UNSET
    if not isinstance(channel, Unset):
        json_channel = channel.value

    params["channel"] = json_channel

    params["paidOnly"] = paid_only

    params["page"] = page

    params["pageSize"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CampaignList | Problem:
    if response.status_code == 200:
        response_200 = CampaignList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CampaignList | Problem]:
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
    status: ListCampaignsStatus | Unset = ListCampaignsStatus.ALL,
    q: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    sender: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    channel: ListCampaignsChannel | Unset = UNSET,
    paid_only: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> Response[CampaignList | Problem]:
    """ List campaigns with status, audience, route and results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListCampaignsStatus | Unset):  Default: ListCampaignsStatus.ALL.
        q (str | Unset): Search names and subjects
        from_ (str | Unset):
        to (str | Unset):
        sender (str | Unset):
        list_ (str | Unset):
        tag (str | Unset):
        channel (ListCampaignsChannel | Unset): Only this channel's campaigns
        paid_only (bool | Unset): Only campaigns sent to paid members only
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
q=q,
from_=from_,
to=to,
sender=sender,
list_=list_,
tag=tag,
channel=channel,
paid_only=paid_only,
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
    status: ListCampaignsStatus | Unset = ListCampaignsStatus.ALL,
    q: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    sender: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    channel: ListCampaignsChannel | Unset = UNSET,
    paid_only: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> CampaignList | Problem | None:
    """ List campaigns with status, audience, route and results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListCampaignsStatus | Unset):  Default: ListCampaignsStatus.ALL.
        q (str | Unset): Search names and subjects
        from_ (str | Unset):
        to (str | Unset):
        sender (str | Unset):
        list_ (str | Unset):
        tag (str | Unset):
        channel (ListCampaignsChannel | Unset): Only this channel's campaigns
        paid_only (bool | Unset): Only campaigns sent to paid members only
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,
q=q,
from_=from_,
to=to,
sender=sender,
list_=list_,
tag=tag,
channel=channel,
paid_only=paid_only,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListCampaignsStatus | Unset = ListCampaignsStatus.ALL,
    q: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    sender: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    channel: ListCampaignsChannel | Unset = UNSET,
    paid_only: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> Response[CampaignList | Problem]:
    """ List campaigns with status, audience, route and results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListCampaignsStatus | Unset):  Default: ListCampaignsStatus.ALL.
        q (str | Unset): Search names and subjects
        from_ (str | Unset):
        to (str | Unset):
        sender (str | Unset):
        list_ (str | Unset):
        tag (str | Unset):
        channel (ListCampaignsChannel | Unset): Only this channel's campaigns
        paid_only (bool | Unset): Only campaigns sent to paid members only
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
q=q,
from_=from_,
to=to,
sender=sender,
list_=list_,
tag=tag,
channel=channel,
paid_only=paid_only,
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
    status: ListCampaignsStatus | Unset = ListCampaignsStatus.ALL,
    q: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    sender: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    channel: ListCampaignsChannel | Unset = UNSET,
    paid_only: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> CampaignList | Problem | None:
    """ List campaigns with status, audience, route and results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListCampaignsStatus | Unset):  Default: ListCampaignsStatus.ALL.
        q (str | Unset): Search names and subjects
        from_ (str | Unset):
        to (str | Unset):
        sender (str | Unset):
        list_ (str | Unset):
        tag (str | Unset):
        channel (ListCampaignsChannel | Unset): Only this channel's campaigns
        paid_only (bool | Unset): Only campaigns sent to paid members only
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,
q=q,
from_=from_,
to=to,
sender=sender,
list_=list_,
tag=tag,
channel=channel,
paid_only=paid_only,
page=page,
page_size=page_size,

    )).parsed
