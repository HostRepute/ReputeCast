from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.report_input import ReportInput
from ...models.visibility_report import VisibilityReport
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    body: ReportInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/visibility".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VisibilityReport:
    if response.status_code == 200:
        response_200 = VisibilityReport.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VisibilityReport]:
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
    body: ReportInput,

) -> Response[Problem | VisibilityReport]:
    """ Count the versions of an email's conditional content

     Reads the Visibility rules of the content sent (the composer's unsaved MJML and HTML) or saved, and
    counts the versions the campaign's audience gets with the live audience. The count is kept until the
    rules or the audience change; recount counts again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VisibilityReport]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
body=body,

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
    body: ReportInput,

) -> Problem | VisibilityReport | None:
    """ Count the versions of an email's conditional content

     Reads the Visibility rules of the content sent (the composer's unsaved MJML and HTML) or saved, and
    counts the versions the campaign's audience gets with the live audience. The count is kept until the
    rules or the audience change; recount counts again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VisibilityReport
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ReportInput,

) -> Response[Problem | VisibilityReport]:
    """ Count the versions of an email's conditional content

     Reads the Visibility rules of the content sent (the composer's unsaved MJML and HTML) or saved, and
    counts the versions the campaign's audience gets with the live audience. The count is kept until the
    rules or the audience change; recount counts again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VisibilityReport]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
body=body,

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
    body: ReportInput,

) -> Problem | VisibilityReport | None:
    """ Count the versions of an email's conditional content

     Reads the Visibility rules of the content sent (the composer's unsaved MJML and HTML) or saved, and
    counts the versions the campaign's audience gets with the live audience. The count is kept until the
    rules or the audience change; recount counts again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VisibilityReport
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
body=body,

    )).parsed
