from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_template import TransactionalTemplate
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    template: str,
    *,
    language: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["language"] = language


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/transactional/templates/{template}".format(workspace=quote(str(workspace), safe=""),template=quote(str(template), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalTemplate:
    if response.status_code == 200:
        response_200 = TransactionalTemplate.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalTemplate]:
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
    language: str | Unset = UNSET,

) -> Response[Problem | TransactionalTemplate]:
    """ Get a template with its draft, variables, test data and live version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): A language version's content in place of the draft's
            (internal/languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalTemplate]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
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
    language: str | Unset = UNSET,

) -> Problem | TransactionalTemplate | None:
    """ Get a template with its draft, variables, test data and live version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): A language version's content in place of the draft's
            (internal/languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalTemplate
     """


    return sync_detailed(
        workspace=workspace,
template=template,
client=client,
language=language,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> Response[Problem | TransactionalTemplate]:
    """ Get a template with its draft, variables, test data and live version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): A language version's content in place of the draft's
            (internal/languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalTemplate]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
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
    language: str | Unset = UNSET,

) -> Problem | TransactionalTemplate | None:
    """ Get a template with its draft, variables, test data and live version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str | Unset): A language version's content in place of the draft's
            (internal/languages)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalTemplate
     """


    return (await asyncio_detailed(
        workspace=workspace,
template=template,
client=client,
language=language,

    )).parsed
