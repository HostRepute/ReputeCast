from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_contact_activity import TransactionalContactActivity
from typing import cast



def _get_kwargs(
    workspace: str,
    contact_id: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/transactional/contacts/{contact_id}".format(workspace=quote(str(workspace), safe=""),contact_id=quote(str(contact_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalContactActivity:
    if response.status_code == 200:
        response_200 = TransactionalContactActivity.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalContactActivity]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    contact_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | TransactionalContactActivity]:
    """ A contact's transactional messages and category opt-outs

     For the contact profile's activity timeline. Content stays behind transactional:content.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalContactActivity]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact_id=contact_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    contact_id: str,
    *,
    client: AuthenticatedClient,

) -> Problem | TransactionalContactActivity | None:
    """ A contact's transactional messages and category opt-outs

     For the contact profile's activity timeline. Content stays behind transactional:content.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalContactActivity
     """


    return sync_detailed(
        workspace=workspace,
contact_id=contact_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | TransactionalContactActivity]:
    """ A contact's transactional messages and category opt-outs

     For the contact profile's activity timeline. Content stays behind transactional:content.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalContactActivity]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact_id=contact_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    contact_id: str,
    *,
    client: AuthenticatedClient,

) -> Problem | TransactionalContactActivity | None:
    """ A contact's transactional messages and category opt-outs

     For the contact profile's activity timeline. Content stays behind transactional:content.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalContactActivity
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact_id=contact_id,
client=client,

    )).parsed
