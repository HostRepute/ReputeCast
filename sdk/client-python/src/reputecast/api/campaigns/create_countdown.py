from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.countdown_image import CountdownImage
from ...models.countdown_settings import CountdownSettings
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: CountdownSettings,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/countdowns".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CountdownImage | Problem:
    if response.status_code == 200:
        response_200 = CountdownImage.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CountdownImage | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: CountdownSettings,

) -> Response[CountdownImage | Problem]:
    """ The image of a Countdown block's settings

     The image is an animated GIF drawn each time the email is opened: up to a minute of the time left,
    then it stays on its last frame. The same settings always return the same image. The digits need a
    contrast of at least 4.5:1 on the background.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (CountdownSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CountdownImage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: CountdownSettings,

) -> CountdownImage | Problem | None:
    """ The image of a Countdown block's settings

     The image is an animated GIF drawn each time the email is opened: up to a minute of the time left,
    then it stays on its last frame. The same settings always return the same image. The digits need a
    contrast of at least 4.5:1 on the background.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (CountdownSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CountdownImage | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: CountdownSettings,

) -> Response[CountdownImage | Problem]:
    """ The image of a Countdown block's settings

     The image is an animated GIF drawn each time the email is opened: up to a minute of the time left,
    then it stays on its last frame. The same settings always return the same image. The digits need a
    contrast of at least 4.5:1 on the background.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (CountdownSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CountdownImage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: CountdownSettings,

) -> CountdownImage | Problem | None:
    """ The image of a Countdown block's settings

     The image is an animated GIF drawn each time the email is opened: up to a minute of the time left,
    then it stays on its last frame. The same settings always return the same image. The digits need a
    contrast of at least 4.5:1 on the background.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (CountdownSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CountdownImage | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
