from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.survey import Survey
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    survey_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/surveys/{survey_id}/publish".format(workspace=quote(str(workspace), safe=""),survey_id=quote(str(survey_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | Survey:
    if response.status_code == 200:
        response_200 = Survey.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | Survey]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    survey_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | Survey]:
    """ Publish a survey: its page goes live (logged)

     Every question must be complete. The page records answers only when someone presses Send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | Survey]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
survey_id=survey_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    survey_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | Survey | None:
    """ Publish a survey: its page goes live (logged)

     Every question must be complete. The page records answers only when someone presses Send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | Survey
     """


    return sync_detailed(
        workspace=workspace,
survey_id=survey_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    survey_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | Survey]:
    """ Publish a survey: its page goes live (logged)

     Every question must be complete. The page records answers only when someone presses Send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | Survey]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
survey_id=survey_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    survey_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | Survey | None:
    """ Publish a survey: its page goes live (logged)

     Every question must be complete. The page records answers only when someone presses Send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | Survey
     """


    return (await asyncio_detailed(
        workspace=workspace,
survey_id=survey_id,
client=client,

    )).parsed
