from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.scoring_impact import ScoringImpact
from ...models.scoring_preview_input import ScoringPreviewInput
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: ScoringPreviewInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/scoring/preview".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ScoringImpact:
    if response.status_code == 200:
        response_200 = ScoringImpact.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ScoringImpact]:
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
    body: ScoringPreviewInput,

) -> Response[Problem | ScoringImpact]:
    """ How an edited, unsaved set-up would move contacts between bands (nothing is saved)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ScoringPreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ScoringImpact]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: ScoringPreviewInput,

) -> Problem | ScoringImpact | None:
    """ How an edited, unsaved set-up would move contacts between bands (nothing is saved)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ScoringPreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ScoringImpact
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: ScoringPreviewInput,

) -> Response[Problem | ScoringImpact]:
    """ How an edited, unsaved set-up would move contacts between bands (nothing is saved)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ScoringPreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ScoringImpact]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: ScoringPreviewInput,

) -> Problem | ScoringImpact | None:
    """ How an edited, unsaved set-up would move contacts between bands (nothing is saved)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ScoringPreviewInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ScoringImpact
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
