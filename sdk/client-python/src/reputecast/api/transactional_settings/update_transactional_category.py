from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_category import TransactionalCategory
from ...models.transactional_category_input import TransactionalCategoryInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    category_id: UUID,
    *,
    body: TransactionalCategoryInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/transactional/categories/{category_id}".format(workspace=quote(str(workspace), safe=""),category_id=quote(str(category_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalCategory:
    if response.status_code == 200:
        response_200 = TransactionalCategory.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalCategory]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    category_id: UUID,
    *,
    client: AuthenticatedClient,
    body: TransactionalCategoryInput,

) -> Response[Problem | TransactionalCategory]:
    """ Change a category (its kind is fixed while it has templates)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        category_id (UUID):
        body (TransactionalCategoryInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalCategory]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
category_id=category_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    category_id: UUID,
    *,
    client: AuthenticatedClient,
    body: TransactionalCategoryInput,

) -> Problem | TransactionalCategory | None:
    """ Change a category (its kind is fixed while it has templates)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        category_id (UUID):
        body (TransactionalCategoryInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalCategory
     """


    return sync_detailed(
        workspace=workspace,
category_id=category_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    category_id: UUID,
    *,
    client: AuthenticatedClient,
    body: TransactionalCategoryInput,

) -> Response[Problem | TransactionalCategory]:
    """ Change a category (its kind is fixed while it has templates)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        category_id (UUID):
        body (TransactionalCategoryInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalCategory]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
category_id=category_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    category_id: UUID,
    *,
    client: AuthenticatedClient,
    body: TransactionalCategoryInput,

) -> Problem | TransactionalCategory | None:
    """ Change a category (its kind is fixed while it has templates)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        category_id (UUID):
        body (TransactionalCategoryInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalCategory
     """


    return (await asyncio_detailed(
        workspace=workspace,
category_id=category_id,
client=client,
body=body,

    )).parsed
