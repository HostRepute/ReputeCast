from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.transactional_content import TransactionalContent
from typing import cast



def _get_kwargs(
    workspace: str,
    message_id: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/transactional/messages/{message_id}/content".format(workspace=quote(str(workspace), safe=""),message_id=quote(str(message_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalContent:
    if response.status_code == 200:
        response_200 = TransactionalContent.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalContent]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | TransactionalContent]:
    """ A message's rendered HTML, text and headers

     Content can hold personal data: it needs transactional:content and every view is audited. Sensitive
    values are masked. Answers 410 content_removed after the content retention.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalContent]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
message_id=message_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,

) -> Problem | TransactionalContent | None:
    """ A message's rendered HTML, text and headers

     Content can hold personal data: it needs transactional:content and every view is audited. Sensitive
    values are masked. Answers 410 content_removed after the content retention.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalContent
     """


    return sync_detailed(
        workspace=workspace,
message_id=message_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | TransactionalContent]:
    """ A message's rendered HTML, text and headers

     Content can hold personal data: it needs transactional:content and every view is audited. Sensitive
    values are masked. Answers 410 content_removed after the content retention.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalContent]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
message_id=message_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    message_id: str,
    *,
    client: AuthenticatedClient,

) -> Problem | TransactionalContent | None:
    """ A message's rendered HTML, text and headers

     Content can hold personal data: it needs transactional:content and every view is audited. Sensitive
    values are masked. Answers 410 content_removed after the content retention.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        message_id (str): The message ID the send API answered

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalContent
     """


    return (await asyncio_detailed(
        workspace=workspace,
message_id=message_id,
client=client,

    )).parsed
