from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.survey_results import SurveyResults
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    survey_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/surveys/{survey_id}/results".format(workspace=quote(str(workspace), safe=""),survey_id=quote(str(survey_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SurveyResults:
    if response.status_code == 200:
        response_200 = SurveyResults.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SurveyResults]:
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

) -> Response[Problem | SurveyResults]:
    """ A survey's completion and per-question results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SurveyResults]
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

) -> Problem | SurveyResults | None:
    """ A survey's completion and per-question results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SurveyResults
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

) -> Response[Problem | SurveyResults]:
    """ A survey's completion and per-question results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SurveyResults]
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

) -> Problem | SurveyResults | None:
    """ A survey's completion and per-question results

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SurveyResults
     """


    return (await asyncio_detailed(
        workspace=workspace,
survey_id=survey_id,
client=client,

    )).parsed
