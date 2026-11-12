from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_campaign_report_bounce_review_type import GetCampaignReportBounceReviewType
from ...models.problem import Problem
from ...models.report_bounce_review import ReportBounceReview
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    type_: GetCampaignReportBounceReviewType,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/report/bounces/{type_}".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),type_=quote(str(type_), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReportBounceReview:
    if response.status_code == 200:
        response_200 = ReportBounceReview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReportBounceReview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    type_: GetCampaignReportBounceReviewType,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReportBounceReview]:
    """ What the providers said for one bounce type (no addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        type_ (GetCampaignReportBounceReviewType): Bounce type

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReportBounceReview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
type_=type_,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    type_: GetCampaignReportBounceReviewType,
    *,
    client: AuthenticatedClient,

) -> Problem | ReportBounceReview | None:
    """ What the providers said for one bounce type (no addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        type_ (GetCampaignReportBounceReviewType): Bounce type

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReportBounceReview
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
type_=type_,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    type_: GetCampaignReportBounceReviewType,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReportBounceReview]:
    """ What the providers said for one bounce type (no addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        type_ (GetCampaignReportBounceReviewType): Bounce type

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReportBounceReview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
type_=type_,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    type_: GetCampaignReportBounceReviewType,
    *,
    client: AuthenticatedClient,

) -> Problem | ReportBounceReview | None:
    """ What the providers said for one bounce type (no addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id
        type_ (GetCampaignReportBounceReviewType): Bounce type

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReportBounceReview
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
type_=type_,
client=client,

    )).parsed
