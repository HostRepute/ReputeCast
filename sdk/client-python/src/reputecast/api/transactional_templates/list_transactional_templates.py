from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_transactional_templates_period import ListTransactionalTemplatesPeriod
from ...models.problem import Problem
from ...models.transactional_template_list import TransactionalTemplateList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    period: ListTransactionalTemplatesPeriod | Unset = ListTransactionalTemplatesPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_period: str | Unset = UNSET
    if not isinstance(period, Unset):
        json_period = period.value

    params["period"] = json_period

    params["includeTest"] = include_test


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/transactional/templates".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TransactionalTemplateList:
    if response.status_code == 200:
        response_200 = TransactionalTemplateList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TransactionalTemplateList]:
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
    period: ListTransactionalTemplatesPeriod | Unset = ListTransactionalTemplatesPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Response[Problem | TransactionalTemplateList]:
    """ List templates with their numbers over a period

     Each template's live version, draft state, sent, delivery, bounce and complaint rates, median
    delivery time, last send and a sparkline. Test messages are left out unless includeTest.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListTransactionalTemplatesPeriod | Unset): The period of the numbers Default:
            ListTransactionalTemplatesPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalTemplateList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
include_test=include_test,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: ListTransactionalTemplatesPeriod | Unset = ListTransactionalTemplatesPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Problem | TransactionalTemplateList | None:
    """ List templates with their numbers over a period

     Each template's live version, draft state, sent, delivery, bounce and complaint rates, median
    delivery time, last send and a sparkline. Test messages are left out unless includeTest.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListTransactionalTemplatesPeriod | Unset): The period of the numbers Default:
            ListTransactionalTemplatesPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalTemplateList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
period=period,
include_test=include_test,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: ListTransactionalTemplatesPeriod | Unset = ListTransactionalTemplatesPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Response[Problem | TransactionalTemplateList]:
    """ List templates with their numbers over a period

     Each template's live version, draft state, sent, delivery, bounce and complaint rates, median
    delivery time, last send and a sparkline. Test messages are left out unless includeTest.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListTransactionalTemplatesPeriod | Unset): The period of the numbers Default:
            ListTransactionalTemplatesPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TransactionalTemplateList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
period=period,
include_test=include_test,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    period: ListTransactionalTemplatesPeriod | Unset = ListTransactionalTemplatesPeriod.VALUE_1,
    include_test: bool | Unset = UNSET,

) -> Problem | TransactionalTemplateList | None:
    """ List templates with their numbers over a period

     Each template's live version, draft state, sent, delivery, bounce and complaint rates, median
    delivery time, last send and a sparkline. Test messages are left out unless includeTest.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        period (ListTransactionalTemplatesPeriod | Unset): The period of the numbers Default:
            ListTransactionalTemplatesPeriod.VALUE_1.
        include_test (bool | Unset): Count messages sent with test-mode keys

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TransactionalTemplateList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
period=period,
include_test=include_test,

    )).parsed
