from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.survey_response_list import SurveyResponseList
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    survey_id: UUID,
    *,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["offset"] = offset

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/surveys/{survey_id}/responses".format(workspace=quote(str(workspace), safe=""),survey_id=quote(str(survey_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SurveyResponseList:
    if response.status_code == 200:
        response_200 = SurveyResponseList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SurveyResponseList]:
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
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Response[Problem | SurveyResponseList]:
    """ Completed responses, newest first

     Contacts are named for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SurveyResponseList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
survey_id=survey_id,
offset=offset,
limit=limit,

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
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Problem | SurveyResponseList | None:
    """ Completed responses, newest first

     Contacts are named for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SurveyResponseList
     """


    return sync_detailed(
        workspace=workspace,
survey_id=survey_id,
client=client,
offset=offset,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    survey_id: UUID,
    *,
    client: AuthenticatedClient,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Response[Problem | SurveyResponseList]:
    """ Completed responses, newest first

     Contacts are named for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SurveyResponseList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
survey_id=survey_id,
offset=offset,
limit=limit,

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
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Problem | SurveyResponseList | None:
    """ Completed responses, newest first

     Contacts are named for people with subscribers:view.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        survey_id (UUID): Survey id
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SurveyResponseList
     """


    return (await asyncio_detailed(
        workspace=workspace,
survey_id=survey_id,
client=client,
offset=offset,
limit=limit,

    )).parsed
