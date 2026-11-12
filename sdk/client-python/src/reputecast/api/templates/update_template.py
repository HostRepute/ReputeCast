from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.email_template_detail import EmailTemplateDetail
from ...models.email_template_input import EmailTemplateInput
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    template_id: UUID,
    *,
    body: EmailTemplateInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/templates/{template_id}".format(workspace=quote(str(workspace), safe=""),template_id=quote(str(template_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmailTemplateDetail | Problem:
    if response.status_code == 200:
        response_200 = EmailTemplateDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EmailTemplateDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    template_id: UUID,
    *,
    client: AuthenticatedClient,
    body: EmailTemplateInput,

) -> Response[EmailTemplateDetail | Problem]:
    """ Rename a template or replace its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (UUID):
        body (EmailTemplateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailTemplateDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template_id=template_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    template_id: UUID,
    *,
    client: AuthenticatedClient,
    body: EmailTemplateInput,

) -> EmailTemplateDetail | Problem | None:
    """ Rename a template or replace its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (UUID):
        body (EmailTemplateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailTemplateDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
template_id=template_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    template_id: UUID,
    *,
    client: AuthenticatedClient,
    body: EmailTemplateInput,

) -> Response[EmailTemplateDetail | Problem]:
    """ Rename a template or replace its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (UUID):
        body (EmailTemplateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailTemplateDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template_id=template_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    template_id: UUID,
    *,
    client: AuthenticatedClient,
    body: EmailTemplateInput,

) -> EmailTemplateDetail | Problem | None:
    """ Rename a template or replace its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (UUID):
        body (EmailTemplateInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailTemplateDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
template_id=template_id,
client=client,
body=body,

    )).parsed
