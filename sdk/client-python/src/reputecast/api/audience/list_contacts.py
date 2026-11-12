from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.contact_list import ContactList
from ...models.list_contacts_band import ListContactsBand
from ...models.list_contacts_check import ListContactsCheck
from ...models.list_contacts_consent import ListContactsConsent
from ...models.list_contacts_mailbox import ListContactsMailbox
from ...models.list_contacts_paid import ListContactsPaid
from ...models.list_contacts_phone import ListContactsPhone
from ...models.list_contacts_sort import ListContactsSort
from ...models.list_contacts_tier import ListContactsTier
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tier: ListContactsTier | Unset = UNSET,
    mailbox: ListContactsMailbox | Unset = UNSET,
    consent: ListContactsConsent | Unset = UNSET,
    source: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    segment: str | Unset = UNSET,
    joined_after: str | Unset = UNSET,
    joined_before: str | Unset = UNSET,
    import_: str | Unset = UNSET,
    check: ListContactsCheck | Unset = UNSET,
    language: str | Unset = UNSET,
    band: ListContactsBand | Unset = UNSET,
    score_from: str | Unset = UNSET,
    score_to: str | Unset = UNSET,
    growth_source: str | Unset = UNSET,
    utm_source: str | Unset = UNSET,
    utm_medium: str | Unset = UNSET,
    utm_campaign: str | Unset = UNSET,
    referring_domain: str | Unset = UNSET,
    sort: ListContactsSort | Unset = UNSET,
    signed_in: bool | Unset = UNSET,
    paid: ListContactsPaid | Unset = UNSET,
    paid_tier: str | Unset = UNSET,
    push_subscribed: bool | Unset = UNSET,
    whats_app_consent: bool | Unset = UNSET,
    bought: str | Unset = UNSET,
    store_customer: str | Unset = UNSET,
    spent_from: str | Unset = UNSET,
    spent_to: str | Unset = UNSET,
    phone: ListContactsPhone | Unset = UNSET,
    phone_country: str | Unset = UNSET,
    sms_consent: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,
    fields: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q

    params["list"] = list_

    json_tier: str | Unset = UNSET
    if not isinstance(tier, Unset):
        json_tier = tier.value

    params["tier"] = json_tier

    json_mailbox: str | Unset = UNSET
    if not isinstance(mailbox, Unset):
        json_mailbox = mailbox.value

    params["mailbox"] = json_mailbox

    json_consent: str | Unset = UNSET
    if not isinstance(consent, Unset):
        json_consent = consent.value

    params["consent"] = json_consent

    params["source"] = source

    params["tag"] = tag

    params["segment"] = segment

    params["joinedAfter"] = joined_after

    params["joinedBefore"] = joined_before

    params["import"] = import_

    json_check: str | Unset = UNSET
    if not isinstance(check, Unset):
        json_check = check.value

    params["check"] = json_check

    params["language"] = language

    json_band: str | Unset = UNSET
    if not isinstance(band, Unset):
        json_band = band.value

    params["band"] = json_band

    params["scoreFrom"] = score_from

    params["scoreTo"] = score_to

    params["growthSource"] = growth_source

    params["utmSource"] = utm_source

    params["utmMedium"] = utm_medium

    params["utmCampaign"] = utm_campaign

    params["referringDomain"] = referring_domain

    json_sort: str | Unset = UNSET
    if not isinstance(sort, Unset):
        json_sort = sort.value

    params["sort"] = json_sort

    params["signedIn"] = signed_in

    json_paid: str | Unset = UNSET
    if not isinstance(paid, Unset):
        json_paid = paid.value

    params["paid"] = json_paid

    params["paidTier"] = paid_tier

    params["pushSubscribed"] = push_subscribed

    params["whatsAppConsent"] = whats_app_consent

    params["bought"] = bought

    params["storeCustomer"] = store_customer

    params["spentFrom"] = spent_from

    params["spentTo"] = spent_to

    json_phone: str | Unset = UNSET
    if not isinstance(phone, Unset):
        json_phone = phone.value

    params["phone"] = json_phone

    params["phoneCountry"] = phone_country

    params["smsConsent"] = sms_consent

    params["cursor"] = cursor

    params["page"] = page

    params["pageSize"] = page_size

    params["exact"] = exact

    params["fields"] = fields


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/contacts".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ContactList | Problem:
    if response.status_code == 200:
        response_200 = ContactList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ContactList | Problem]:
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
    q: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tier: ListContactsTier | Unset = UNSET,
    mailbox: ListContactsMailbox | Unset = UNSET,
    consent: ListContactsConsent | Unset = UNSET,
    source: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    segment: str | Unset = UNSET,
    joined_after: str | Unset = UNSET,
    joined_before: str | Unset = UNSET,
    import_: str | Unset = UNSET,
    check: ListContactsCheck | Unset = UNSET,
    language: str | Unset = UNSET,
    band: ListContactsBand | Unset = UNSET,
    score_from: str | Unset = UNSET,
    score_to: str | Unset = UNSET,
    growth_source: str | Unset = UNSET,
    utm_source: str | Unset = UNSET,
    utm_medium: str | Unset = UNSET,
    utm_campaign: str | Unset = UNSET,
    referring_domain: str | Unset = UNSET,
    sort: ListContactsSort | Unset = UNSET,
    signed_in: bool | Unset = UNSET,
    paid: ListContactsPaid | Unset = UNSET,
    paid_tier: str | Unset = UNSET,
    push_subscribed: bool | Unset = UNSET,
    whats_app_consent: bool | Unset = UNSET,
    bought: str | Unset = UNSET,
    store_customer: str | Unset = UNSET,
    spent_from: str | Unset = UNSET,
    spent_to: str | Unset = UNSET,
    phone: ListContactsPhone | Unset = UNSET,
    phone_country: str | Unset = UNSET,
    sms_consent: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,
    fields: str | Unset = UNSET,

) -> Response[ContactList | Problem]:
    """ List contacts, most recent activity first (or the order asked for)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Email, name or tag
        list_ (str | Unset): List id
        tier (ListContactsTier | Unset):
        mailbox (ListContactsMailbox | Unset):
        consent (ListContactsConsent | Unset):
        source (str | Unset):
        tag (str | Unset):
        segment (str | Unset): Segment id
        joined_after (str | Unset): Joined on or after this day (2026-09-01), in the workspace's
            time zone
        joined_before (str | Unset): Joined on or before this day, in the workspace's time zone
        import_ (str | Unset): An import's id: the contacts it brought in
        check (ListContactsCheck | Unset): The latest address check's result
        language (str | Unset): A workspace language's code (es also matches es-MX), none or
            not_offered
        band (ListContactsBand | Unset):
        score_from (str | Unset):
        score_to (str | Unset):
        growth_source (str | Unset):
        utm_source (str | Unset):
        utm_medium (str | Unset):
        utm_campaign (str | Unset):
        referring_domain (str | Unset):
        sort (ListContactsSort | Unset):
        signed_in (bool | Unset): Only contacts who signed in to the website
        paid (ListContactsPaid | Unset):
        paid_tier (str | Unset): With paid=with_access: a tier's id, for that tier and above
        push_subscribed (bool | Unset): Only contacts with a linked push subscription
        whats_app_consent (bool | Unset): Only contacts with WhatsApp consent on a list
        bought (str | Unset): Only contacts who bought a product: any, or the product's id
        store_customer (str | Unset): A store's id, or any
        spent_from (str | Unset): Total spent in the stores, at least
        spent_to (str | Unset): Total spent in the stores, at most
        phone (ListContactsPhone | Unset): yes: contacts with a phone number; no: without one
        phone_country (str | Unset): The phone number's country (GB)
        sms_consent (str | Unset): any: SMS consent on some list; none: on none; or a list id
        cursor (str | Unset): nextCursor or prevCursor of the page before; pages in constant time
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000 (slower on large audiences)
        fields (str | Unset): Custom field keys, comma-separated (at most 10), whose values each
            row carries

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
list_=list_,
tier=tier,
mailbox=mailbox,
consent=consent,
source=source,
tag=tag,
segment=segment,
joined_after=joined_after,
joined_before=joined_before,
import_=import_,
check=check,
language=language,
band=band,
score_from=score_from,
score_to=score_to,
growth_source=growth_source,
utm_source=utm_source,
utm_medium=utm_medium,
utm_campaign=utm_campaign,
referring_domain=referring_domain,
sort=sort,
signed_in=signed_in,
paid=paid,
paid_tier=paid_tier,
push_subscribed=push_subscribed,
whats_app_consent=whats_app_consent,
bought=bought,
store_customer=store_customer,
spent_from=spent_from,
spent_to=spent_to,
phone=phone,
phone_country=phone_country,
sms_consent=sms_consent,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,
fields=fields,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tier: ListContactsTier | Unset = UNSET,
    mailbox: ListContactsMailbox | Unset = UNSET,
    consent: ListContactsConsent | Unset = UNSET,
    source: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    segment: str | Unset = UNSET,
    joined_after: str | Unset = UNSET,
    joined_before: str | Unset = UNSET,
    import_: str | Unset = UNSET,
    check: ListContactsCheck | Unset = UNSET,
    language: str | Unset = UNSET,
    band: ListContactsBand | Unset = UNSET,
    score_from: str | Unset = UNSET,
    score_to: str | Unset = UNSET,
    growth_source: str | Unset = UNSET,
    utm_source: str | Unset = UNSET,
    utm_medium: str | Unset = UNSET,
    utm_campaign: str | Unset = UNSET,
    referring_domain: str | Unset = UNSET,
    sort: ListContactsSort | Unset = UNSET,
    signed_in: bool | Unset = UNSET,
    paid: ListContactsPaid | Unset = UNSET,
    paid_tier: str | Unset = UNSET,
    push_subscribed: bool | Unset = UNSET,
    whats_app_consent: bool | Unset = UNSET,
    bought: str | Unset = UNSET,
    store_customer: str | Unset = UNSET,
    spent_from: str | Unset = UNSET,
    spent_to: str | Unset = UNSET,
    phone: ListContactsPhone | Unset = UNSET,
    phone_country: str | Unset = UNSET,
    sms_consent: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,
    fields: str | Unset = UNSET,

) -> ContactList | Problem | None:
    """ List contacts, most recent activity first (or the order asked for)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Email, name or tag
        list_ (str | Unset): List id
        tier (ListContactsTier | Unset):
        mailbox (ListContactsMailbox | Unset):
        consent (ListContactsConsent | Unset):
        source (str | Unset):
        tag (str | Unset):
        segment (str | Unset): Segment id
        joined_after (str | Unset): Joined on or after this day (2026-09-01), in the workspace's
            time zone
        joined_before (str | Unset): Joined on or before this day, in the workspace's time zone
        import_ (str | Unset): An import's id: the contacts it brought in
        check (ListContactsCheck | Unset): The latest address check's result
        language (str | Unset): A workspace language's code (es also matches es-MX), none or
            not_offered
        band (ListContactsBand | Unset):
        score_from (str | Unset):
        score_to (str | Unset):
        growth_source (str | Unset):
        utm_source (str | Unset):
        utm_medium (str | Unset):
        utm_campaign (str | Unset):
        referring_domain (str | Unset):
        sort (ListContactsSort | Unset):
        signed_in (bool | Unset): Only contacts who signed in to the website
        paid (ListContactsPaid | Unset):
        paid_tier (str | Unset): With paid=with_access: a tier's id, for that tier and above
        push_subscribed (bool | Unset): Only contacts with a linked push subscription
        whats_app_consent (bool | Unset): Only contacts with WhatsApp consent on a list
        bought (str | Unset): Only contacts who bought a product: any, or the product's id
        store_customer (str | Unset): A store's id, or any
        spent_from (str | Unset): Total spent in the stores, at least
        spent_to (str | Unset): Total spent in the stores, at most
        phone (ListContactsPhone | Unset): yes: contacts with a phone number; no: without one
        phone_country (str | Unset): The phone number's country (GB)
        sms_consent (str | Unset): any: SMS consent on some list; none: on none; or a list id
        cursor (str | Unset): nextCursor or prevCursor of the page before; pages in constant time
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000 (slower on large audiences)
        fields (str | Unset): Custom field keys, comma-separated (at most 10), whose values each
            row carries

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,
list_=list_,
tier=tier,
mailbox=mailbox,
consent=consent,
source=source,
tag=tag,
segment=segment,
joined_after=joined_after,
joined_before=joined_before,
import_=import_,
check=check,
language=language,
band=band,
score_from=score_from,
score_to=score_to,
growth_source=growth_source,
utm_source=utm_source,
utm_medium=utm_medium,
utm_campaign=utm_campaign,
referring_domain=referring_domain,
sort=sort,
signed_in=signed_in,
paid=paid,
paid_tier=paid_tier,
push_subscribed=push_subscribed,
whats_app_consent=whats_app_consent,
bought=bought,
store_customer=store_customer,
spent_from=spent_from,
spent_to=spent_to,
phone=phone,
phone_country=phone_country,
sms_consent=sms_consent,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,
fields=fields,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tier: ListContactsTier | Unset = UNSET,
    mailbox: ListContactsMailbox | Unset = UNSET,
    consent: ListContactsConsent | Unset = UNSET,
    source: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    segment: str | Unset = UNSET,
    joined_after: str | Unset = UNSET,
    joined_before: str | Unset = UNSET,
    import_: str | Unset = UNSET,
    check: ListContactsCheck | Unset = UNSET,
    language: str | Unset = UNSET,
    band: ListContactsBand | Unset = UNSET,
    score_from: str | Unset = UNSET,
    score_to: str | Unset = UNSET,
    growth_source: str | Unset = UNSET,
    utm_source: str | Unset = UNSET,
    utm_medium: str | Unset = UNSET,
    utm_campaign: str | Unset = UNSET,
    referring_domain: str | Unset = UNSET,
    sort: ListContactsSort | Unset = UNSET,
    signed_in: bool | Unset = UNSET,
    paid: ListContactsPaid | Unset = UNSET,
    paid_tier: str | Unset = UNSET,
    push_subscribed: bool | Unset = UNSET,
    whats_app_consent: bool | Unset = UNSET,
    bought: str | Unset = UNSET,
    store_customer: str | Unset = UNSET,
    spent_from: str | Unset = UNSET,
    spent_to: str | Unset = UNSET,
    phone: ListContactsPhone | Unset = UNSET,
    phone_country: str | Unset = UNSET,
    sms_consent: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,
    fields: str | Unset = UNSET,

) -> Response[ContactList | Problem]:
    """ List contacts, most recent activity first (or the order asked for)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Email, name or tag
        list_ (str | Unset): List id
        tier (ListContactsTier | Unset):
        mailbox (ListContactsMailbox | Unset):
        consent (ListContactsConsent | Unset):
        source (str | Unset):
        tag (str | Unset):
        segment (str | Unset): Segment id
        joined_after (str | Unset): Joined on or after this day (2026-09-01), in the workspace's
            time zone
        joined_before (str | Unset): Joined on or before this day, in the workspace's time zone
        import_ (str | Unset): An import's id: the contacts it brought in
        check (ListContactsCheck | Unset): The latest address check's result
        language (str | Unset): A workspace language's code (es also matches es-MX), none or
            not_offered
        band (ListContactsBand | Unset):
        score_from (str | Unset):
        score_to (str | Unset):
        growth_source (str | Unset):
        utm_source (str | Unset):
        utm_medium (str | Unset):
        utm_campaign (str | Unset):
        referring_domain (str | Unset):
        sort (ListContactsSort | Unset):
        signed_in (bool | Unset): Only contacts who signed in to the website
        paid (ListContactsPaid | Unset):
        paid_tier (str | Unset): With paid=with_access: a tier's id, for that tier and above
        push_subscribed (bool | Unset): Only contacts with a linked push subscription
        whats_app_consent (bool | Unset): Only contacts with WhatsApp consent on a list
        bought (str | Unset): Only contacts who bought a product: any, or the product's id
        store_customer (str | Unset): A store's id, or any
        spent_from (str | Unset): Total spent in the stores, at least
        spent_to (str | Unset): Total spent in the stores, at most
        phone (ListContactsPhone | Unset): yes: contacts with a phone number; no: without one
        phone_country (str | Unset): The phone number's country (GB)
        sms_consent (str | Unset): any: SMS consent on some list; none: on none; or a list id
        cursor (str | Unset): nextCursor or prevCursor of the page before; pages in constant time
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000 (slower on large audiences)
        fields (str | Unset): Custom field keys, comma-separated (at most 10), whose values each
            row carries

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
list_=list_,
tier=tier,
mailbox=mailbox,
consent=consent,
source=source,
tag=tag,
segment=segment,
joined_after=joined_after,
joined_before=joined_before,
import_=import_,
check=check,
language=language,
band=band,
score_from=score_from,
score_to=score_to,
growth_source=growth_source,
utm_source=utm_source,
utm_medium=utm_medium,
utm_campaign=utm_campaign,
referring_domain=referring_domain,
sort=sort,
signed_in=signed_in,
paid=paid,
paid_tier=paid_tier,
push_subscribed=push_subscribed,
whats_app_consent=whats_app_consent,
bought=bought,
store_customer=store_customer,
spent_from=spent_from,
spent_to=spent_to,
phone=phone,
phone_country=phone_country,
sms_consent=sms_consent,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,
fields=fields,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    list_: str | Unset = UNSET,
    tier: ListContactsTier | Unset = UNSET,
    mailbox: ListContactsMailbox | Unset = UNSET,
    consent: ListContactsConsent | Unset = UNSET,
    source: str | Unset = UNSET,
    tag: str | Unset = UNSET,
    segment: str | Unset = UNSET,
    joined_after: str | Unset = UNSET,
    joined_before: str | Unset = UNSET,
    import_: str | Unset = UNSET,
    check: ListContactsCheck | Unset = UNSET,
    language: str | Unset = UNSET,
    band: ListContactsBand | Unset = UNSET,
    score_from: str | Unset = UNSET,
    score_to: str | Unset = UNSET,
    growth_source: str | Unset = UNSET,
    utm_source: str | Unset = UNSET,
    utm_medium: str | Unset = UNSET,
    utm_campaign: str | Unset = UNSET,
    referring_domain: str | Unset = UNSET,
    sort: ListContactsSort | Unset = UNSET,
    signed_in: bool | Unset = UNSET,
    paid: ListContactsPaid | Unset = UNSET,
    paid_tier: str | Unset = UNSET,
    push_subscribed: bool | Unset = UNSET,
    whats_app_consent: bool | Unset = UNSET,
    bought: str | Unset = UNSET,
    store_customer: str | Unset = UNSET,
    spent_from: str | Unset = UNSET,
    spent_to: str | Unset = UNSET,
    phone: ListContactsPhone | Unset = UNSET,
    phone_country: str | Unset = UNSET,
    sms_consent: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,
    fields: str | Unset = UNSET,

) -> ContactList | Problem | None:
    """ List contacts, most recent activity first (or the order asked for)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Email, name or tag
        list_ (str | Unset): List id
        tier (ListContactsTier | Unset):
        mailbox (ListContactsMailbox | Unset):
        consent (ListContactsConsent | Unset):
        source (str | Unset):
        tag (str | Unset):
        segment (str | Unset): Segment id
        joined_after (str | Unset): Joined on or after this day (2026-09-01), in the workspace's
            time zone
        joined_before (str | Unset): Joined on or before this day, in the workspace's time zone
        import_ (str | Unset): An import's id: the contacts it brought in
        check (ListContactsCheck | Unset): The latest address check's result
        language (str | Unset): A workspace language's code (es also matches es-MX), none or
            not_offered
        band (ListContactsBand | Unset):
        score_from (str | Unset):
        score_to (str | Unset):
        growth_source (str | Unset):
        utm_source (str | Unset):
        utm_medium (str | Unset):
        utm_campaign (str | Unset):
        referring_domain (str | Unset):
        sort (ListContactsSort | Unset):
        signed_in (bool | Unset): Only contacts who signed in to the website
        paid (ListContactsPaid | Unset):
        paid_tier (str | Unset): With paid=with_access: a tier's id, for that tier and above
        push_subscribed (bool | Unset): Only contacts with a linked push subscription
        whats_app_consent (bool | Unset): Only contacts with WhatsApp consent on a list
        bought (str | Unset): Only contacts who bought a product: any, or the product's id
        store_customer (str | Unset): A store's id, or any
        spent_from (str | Unset): Total spent in the stores, at least
        spent_to (str | Unset): Total spent in the stores, at most
        phone (ListContactsPhone | Unset): yes: contacts with a phone number; no: without one
        phone_country (str | Unset): The phone number's country (GB)
        sms_consent (str | Unset): any: SMS consent on some list; none: on none; or a list id
        cursor (str | Unset): nextCursor or prevCursor of the page before; pages in constant time
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000 (slower on large audiences)
        fields (str | Unset): Custom field keys, comma-separated (at most 10), whose values each
            row carries

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,
list_=list_,
tier=tier,
mailbox=mailbox,
consent=consent,
source=source,
tag=tag,
segment=segment,
joined_after=joined_after,
joined_before=joined_before,
import_=import_,
check=check,
language=language,
band=band,
score_from=score_from,
score_to=score_to,
growth_source=growth_source,
utm_source=utm_source,
utm_medium=utm_medium,
utm_campaign=utm_campaign,
referring_domain=referring_domain,
sort=sort,
signed_in=signed_in,
paid=paid,
paid_tier=paid_tier,
push_subscribed=push_subscribed,
whats_app_consent=whats_app_consent,
bought=bought,
store_customer=store_customer,
spent_from=spent_from,
spent_to=spent_to,
phone=phone,
phone_country=phone_country,
sms_consent=sms_consent,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,
fields=fields,

    )).parsed
