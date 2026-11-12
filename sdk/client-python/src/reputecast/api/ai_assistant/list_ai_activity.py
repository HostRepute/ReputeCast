from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.ai_activity_list import AIActivityList
from ...models.list_ai_activity_feature import ListAiActivityFeature
from ...models.list_ai_activity_outcome import ListAiActivityOutcome
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    member: str | Unset = UNSET,
    feature: ListAiActivityFeature | Unset = UNSET,
    outcome: ListAiActivityOutcome | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["member"] = member

    json_feature: str | Unset = UNSET
    if not isinstance(feature, Unset):
        json_feature = feature.value

    params["feature"] = json_feature

    json_outcome: str | Unset = UNSET
    if not isinstance(outcome, Unset):
        json_outcome = outcome.value

    params["outcome"] = json_outcome

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/ai/activity".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AIActivityList | Problem:
    if response.status_code == 200:
        response_200 = AIActivityList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AIActivityList | Problem]:
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
    member: str | Unset = UNSET,
    feature: ListAiActivityFeature | Unset = UNSET,
    outcome: ListAiActivityOutcome | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[AIActivityList | Problem]:
    """ The AI activity log: each request of the last 90 days, never its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        member (str | Unset): A member's id
        feature (ListAiActivityFeature | Unset):
        outcome (ListAiActivityOutcome | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AIActivityList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
member=member,
feature=feature,
outcome=outcome,
cursor=cursor,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    member: str | Unset = UNSET,
    feature: ListAiActivityFeature | Unset = UNSET,
    outcome: ListAiActivityOutcome | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> AIActivityList | Problem | None:
    """ The AI activity log: each request of the last 90 days, never its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        member (str | Unset): A member's id
        feature (ListAiActivityFeature | Unset):
        outcome (ListAiActivityOutcome | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AIActivityList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
member=member,
feature=feature,
outcome=outcome,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    member: str | Unset = UNSET,
    feature: ListAiActivityFeature | Unset = UNSET,
    outcome: ListAiActivityOutcome | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[AIActivityList | Problem]:
    """ The AI activity log: each request of the last 90 days, never its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        member (str | Unset): A member's id
        feature (ListAiActivityFeature | Unset):
        outcome (ListAiActivityOutcome | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AIActivityList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
member=member,
feature=feature,
outcome=outcome,
cursor=cursor,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    member: str | Unset = UNSET,
    feature: ListAiActivityFeature | Unset = UNSET,
    outcome: ListAiActivityOutcome | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> AIActivityList | Problem | None:
    """ The AI activity log: each request of the last 90 days, never its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        member (str | Unset): A member's id
        feature (ListAiActivityFeature | Unset):
        outcome (ListAiActivityOutcome | Unset):
        cursor (str | Unset):
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AIActivityList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
member=member,
feature=feature,
outcome=outcome,
cursor=cursor,
limit=limit,

    )).parsed
