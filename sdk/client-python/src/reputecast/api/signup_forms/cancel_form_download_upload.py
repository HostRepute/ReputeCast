from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form: UUID,
    upload: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}/download/uploads/{upload}".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),upload=quote(str(upload), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Problem:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Any | Problem]:
    """ Cancel an upload; nothing is kept

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
upload=upload,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Any | Problem | None:
    """ Cancel an upload; nothing is kept

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return sync_detailed(
        workspace=workspace,
form=form,
upload=upload,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Any | Problem]:
    """ Cancel an upload; nothing is kept

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
upload=upload,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Any | Problem | None:
    """ Cancel an upload; nothing is kept

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
form=form,
upload=upload,
client=client,

    )).parsed
