from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_referral_review_signal import GetReferralReviewSignal
from ...models.problem import Problem
from ...models.referral_review_queue import ReferralReviewQueue
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    program: UUID,
    *,
    signal: GetReferralReviewSignal | Unset = UNSET,
    referrer: UUID | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_signal: str | Unset = UNSET
    if not isinstance(signal, Unset):
        json_signal = signal.value

    params["signal"] = json_signal

    json_referrer: str | Unset = UNSET
    if not isinstance(referrer, Unset):
        json_referrer = str(referrer)
    params["referrer"] = json_referrer


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/referrals/{program}/review".format(workspace=quote(str(workspace), safe=""),program=quote(str(program), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReferralReviewQueue:
    if response.status_code == 200:
        response_200 = ReferralReviewQueue.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReferralReviewQueue]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    signal: GetReferralReviewSignal | Unset = UNSET,
    referrer: UUID | Unset = UNSET,

) -> Response[Problem | ReferralReviewQueue]:
    """ Held referrals grouped by referrer, with their evidence

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        signal (GetReferralReviewSignal | Unset):
        referrer (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralReviewQueue]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
signal=signal,
referrer=referrer,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    signal: GetReferralReviewSignal | Unset = UNSET,
    referrer: UUID | Unset = UNSET,

) -> Problem | ReferralReviewQueue | None:
    """ Held referrals grouped by referrer, with their evidence

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        signal (GetReferralReviewSignal | Unset):
        referrer (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralReviewQueue
     """


    return sync_detailed(
        workspace=workspace,
program=program,
client=client,
signal=signal,
referrer=referrer,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    signal: GetReferralReviewSignal | Unset = UNSET,
    referrer: UUID | Unset = UNSET,

) -> Response[Problem | ReferralReviewQueue]:
    """ Held referrals grouped by referrer, with their evidence

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        signal (GetReferralReviewSignal | Unset):
        referrer (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralReviewQueue]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
signal=signal,
referrer=referrer,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    signal: GetReferralReviewSignal | Unset = UNSET,
    referrer: UUID | Unset = UNSET,

) -> Problem | ReferralReviewQueue | None:
    """ Held referrals grouped by referrer, with their evidence

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        signal (GetReferralReviewSignal | Unset):
        referrer (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralReviewQueue
     """


    return (await asyncio_detailed(
        workspace=workspace,
program=program,
client=client,
signal=signal,
referrer=referrer,

    )).parsed
