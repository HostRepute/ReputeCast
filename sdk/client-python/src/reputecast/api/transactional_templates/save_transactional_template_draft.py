from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_draft_save import TransactionalDraftSave
from ...models.transactional_draft_saved import TransactionalDraftSaved
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    template: str,
    *,
    body: TransactionalDraftSave,
    language: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    params: dict[str, Any] = {}

    params["language"] = language


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/transactional/templates/{template}/draft".format(workspace=quote(str(workspace), safe=""),template=quote(str(template), safe=""),),
        "params": params,
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalDraftSaved:
    if response.status_code == 200:
        response_200 = TransactionalDraftSaved.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalDraftSaved]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalDraftSave,
    language: str | Unset = UNSET,

) -> Response[Problem | TransactionalDraftSaved]:
    """ Save the draft (autosave)

     Content, variables, test data, sender, reply-to and tracking. Fails with 409 version_conflict when
    someone saved a newer revision since baseRevision; the problem's meta says who, when and what each
    side changed. Send force: true to keep yours.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): Saves a language version instead of the draft (internal/languages)
        body (TransactionalDraftSave):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalDraftSaved]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
body=body,
language=language,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalDraftSave,
    language: str | Unset = UNSET,

) -> Problem | TransactionalDraftSaved | None:
    """ Save the draft (autosave)

     Content, variables, test data, sender, reply-to and tracking. Fails with 409 version_conflict when
    someone saved a newer revision since baseRevision; the problem's meta says who, when and what each
    side changed. Send force: true to keep yours.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): Saves a language version instead of the draft (internal/languages)
        body (TransactionalDraftSave):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalDraftSaved
     """


    return sync_detailed(
        workspace=workspace,
template=template,
client=client,
body=body,
language=language,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalDraftSave,
    language: str | Unset = UNSET,

) -> Response[Problem | TransactionalDraftSaved]:
    """ Save the draft (autosave)

     Content, variables, test data, sender, reply-to and tracking. Fails with 409 version_conflict when
    someone saved a newer revision since baseRevision; the problem's meta says who, when and what each
    side changed. Send force: true to keep yours.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): Saves a language version instead of the draft (internal/languages)
        body (TransactionalDraftSave):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalDraftSaved]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
body=body,
language=language,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalDraftSave,
    language: str | Unset = UNSET,

) -> Problem | TransactionalDraftSaved | None:
    """ Save the draft (autosave)

     Content, variables, test data, sender, reply-to and tracking. Fails with 409 version_conflict when
    someone saved a newer revision since baseRevision; the problem's meta says who, when and what each
    side changed. Send force: true to keep yours.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): Saves a language version instead of the draft (internal/languages)
        body (TransactionalDraftSave):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalDraftSaved
     """


    return (await asyncio_detailed(
        workspace=workspace,
template=template,
client=client,
body=body,
language=language,

    )).parsed
