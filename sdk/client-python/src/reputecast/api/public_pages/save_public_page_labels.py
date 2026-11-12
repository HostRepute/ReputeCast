from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.public_page_labels import PublicPageLabels
from ...models.public_page_labels_input import PublicPageLabelsInput
from typing import cast



def _get_kwargs(
    workspace: str,
    language: str,
    *,
    body: PublicPageLabelsInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/public-pages/labels/{language}".format(workspace=quote(str(workspace), safe=""),language=quote(str(language), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | PublicPageLabels:
    if response.status_code == 200:
        response_200 = PublicPageLabels.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | PublicPageLabels]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    language: str,
    *,
    client: AuthenticatedClient,
    body: PublicPageLabelsInput,

) -> Response[Problem | PublicPageLabels]:
    """ Save the built-in labels the workspace writes for a language

     Each text may use only its English label's placeholders and tags. Languages ReputeCast ships answer
    409.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str): A workspace language (es)
        body (PublicPageLabelsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPageLabels]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
language=language,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    language: str,
    *,
    client: AuthenticatedClient,
    body: PublicPageLabelsInput,

) -> Problem | PublicPageLabels | None:
    """ Save the built-in labels the workspace writes for a language

     Each text may use only its English label's placeholders and tags. Languages ReputeCast ships answer
    409.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str): A workspace language (es)
        body (PublicPageLabelsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPageLabels
     """


    return sync_detailed(
        workspace=workspace,
language=language,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    language: str,
    *,
    client: AuthenticatedClient,
    body: PublicPageLabelsInput,

) -> Response[Problem | PublicPageLabels]:
    """ Save the built-in labels the workspace writes for a language

     Each text may use only its English label's placeholders and tags. Languages ReputeCast ships answer
    409.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str): A workspace language (es)
        body (PublicPageLabelsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PublicPageLabels]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
language=language,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    language: str,
    *,
    client: AuthenticatedClient,
    body: PublicPageLabelsInput,

) -> Problem | PublicPageLabels | None:
    """ Save the built-in labels the workspace writes for a language

     Each text may use only its English label's placeholders and tags. Languages ReputeCast ships answer
    409.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        language (str): A workspace language (es)
        body (PublicPageLabelsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PublicPageLabels
     """


    return (await asyncio_detailed(
        workspace=workspace,
language=language,
client=client,
body=body,

    )).parsed
