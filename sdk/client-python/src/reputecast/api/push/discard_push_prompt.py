from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.push_prompt import PushPrompt
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/push/prompts/{form_id}/discard".format(workspace=quote(str(workspace), safe=""),form_id=quote(str(form_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | PushPrompt:
    if response.status_code == 200:
        response_200 = PushPrompt.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | PushPrompt]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    form_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | PushPrompt]:
    """ Throw the draft away

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID): Push prompt (form) id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushPrompt]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form_id=form_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    form_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | PushPrompt | None:
    """ Throw the draft away

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID): Push prompt (form) id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PushPrompt
     """


    return sync_detailed(
        workspace=workspace,
form_id=form_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | PushPrompt]:
    """ Throw the draft away

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID): Push prompt (form) id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushPrompt]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form_id=form_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    form_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | PushPrompt | None:
    """ Throw the draft away

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID): Push prompt (form) id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PushPrompt
     """


    return (await asyncio_detailed(
        workspace=workspace,
form_id=form_id,
client=client,

    )).parsed
