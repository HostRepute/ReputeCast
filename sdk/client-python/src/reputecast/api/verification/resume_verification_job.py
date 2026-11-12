from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.verification_job_view import VerificationJobView
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    job: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/verification/jobs/{job}/resume".format(workspace=quote(str(workspace), safe=""),job=quote(str(job), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VerificationJobView:
    if response.status_code == 200:
        response_200 = VerificationJobView.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VerificationJobView]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    job: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | VerificationJobView]:
    """ Continue a job that stopped for credits or an outage

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        job (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationJobView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
job=job,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    job: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | VerificationJobView | None:
    """ Continue a job that stopped for credits or an outage

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        job (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationJobView
     """


    return sync_detailed(
        workspace=workspace,
job=job,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    job: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | VerificationJobView]:
    """ Continue a job that stopped for credits or an outage

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        job (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationJobView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
job=job,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    job: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | VerificationJobView | None:
    """ Continue a job that stopped for credits or an outage

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        job (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationJobView
     """


    return (await asyncio_detailed(
        workspace=workspace,
job=job,
client=client,

    )).parsed
