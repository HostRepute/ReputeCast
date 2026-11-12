from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.push_prompt import PushPrompt
from ...models.push_prompt_save import PushPromptSave
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form_id: UUID,
    *,
    body: PushPromptSave,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/push/prompts/{form_id}".format(workspace=quote(str(workspace), safe=""),form_id=quote(str(form_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
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
    body: PushPromptSave,

) -> Response[Problem | PushPrompt]:
    """ Save the prompt's draft

     Readers see it once it's published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID):
        body (PushPromptSave):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushPrompt]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form_id=form_id,
body=body,

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
    body: PushPromptSave,

) -> Problem | PushPrompt | None:
    """ Save the prompt's draft

     Readers see it once it's published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID):
        body (PushPromptSave):

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
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form_id: UUID,
    *,
    client: AuthenticatedClient,
    body: PushPromptSave,

) -> Response[Problem | PushPrompt]:
    """ Save the prompt's draft

     Readers see it once it's published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID):
        body (PushPromptSave):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushPrompt]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form_id=form_id,
body=body,

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
    body: PushPromptSave,

) -> Problem | PushPrompt | None:
    """ Save the prompt's draft

     Readers see it once it's published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (UUID):
        body (PushPromptSave):

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
body=body,

    )).parsed
