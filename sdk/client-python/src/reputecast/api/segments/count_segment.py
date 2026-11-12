from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.segment_count_result import SegmentCountResult
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    segment: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/segments/{segment}/count".format(workspace=quote(str(workspace), safe=""),segment=quote(str(segment), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SegmentCountResult:
    if response.status_code == 200:
        response_200 = SegmentCountResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SegmentCountResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    segment: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SegmentCountResult]:
    """ Count a saved segment again and store the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        segment (UUID): Segment id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SegmentCountResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
segment=segment,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    segment: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | SegmentCountResult | None:
    """ Count a saved segment again and store the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        segment (UUID): Segment id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SegmentCountResult
     """


    return sync_detailed(
        workspace=workspace,
segment=segment,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    segment: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SegmentCountResult]:
    """ Count a saved segment again and store the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        segment (UUID): Segment id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SegmentCountResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
segment=segment,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    segment: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | SegmentCountResult | None:
    """ Count a saved segment again and store the result

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        segment (UUID): Segment id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SegmentCountResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
segment=segment,
client=client,

    )).parsed
