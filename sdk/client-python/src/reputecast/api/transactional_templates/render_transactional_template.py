from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_render_input import TransactionalRenderInput
from ...models.transactional_rendered import TransactionalRendered
from typing import cast



def _get_kwargs(
    workspace: str,
    template: str,
    *,
    body: TransactionalRenderInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/transactional/templates/{template}/render".format(workspace=quote(str(workspace), safe=""),template=quote(str(template), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalRendered:
    if response.status_code == 200:
        response_200 = TransactionalRendered.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalRendered]:
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
    body: TransactionalRenderInput,

) -> Response[Problem | TransactionalRendered]:
    """ Render the live version with a call's data, without sending

     Returns the subject, HTML and text a send would use. Errors: template_not_found,
    template_not_published, missing_variables (listing them in missing).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (TransactionalRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalRendered]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
body=body,

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
    body: TransactionalRenderInput,

) -> Problem | TransactionalRendered | None:
    """ Render the live version with a call's data, without sending

     Returns the subject, HTML and text a send would use. Errors: template_not_found,
    template_not_published, missing_variables (listing them in missing).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (TransactionalRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalRendered
     """


    return sync_detailed(
        workspace=workspace,
template=template,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: TransactionalRenderInput,

) -> Response[Problem | TransactionalRendered]:
    """ Render the live version with a call's data, without sending

     Returns the subject, HTML and text a send would use. Errors: template_not_found,
    template_not_published, missing_variables (listing them in missing).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (TransactionalRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalRendered]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
body=body,

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
    body: TransactionalRenderInput,

) -> Problem | TransactionalRendered | None:
    """ Render the live version with a call's data, without sending

     Returns the subject, HTML and text a send would use. Errors: template_not_found,
    template_not_published, missing_variables (listing them in missing).

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (TransactionalRenderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalRendered
     """


    return (await asyncio_detailed(
        workspace=workspace,
template=template,
client=client,
body=body,

    )).parsed
