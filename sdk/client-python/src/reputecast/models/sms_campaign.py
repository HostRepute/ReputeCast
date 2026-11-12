from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_campaign_schedule_mode import SmsCampaignScheduleMode
from ..models.sms_campaign_short_domain_status import SmsCampaignShortDomainStatus
from ..models.sms_campaign_status import SmsCampaignStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.country_line import CountryLine
  from ..models.sms_audience_ref import SmsAudienceRef
  from ..models.sms_budget_view import SmsBudgetView
  from ..models.sms_campaign_access import SmsCampaignAccess
  from ..models.sms_counter import SmsCounter
  from ..models.sms_funnel import SmsFunnel
  from ..models.sms_link_view import SmsLinkView
  from ..models.sms_route_line import SmsRouteLine
  from ..models.sms_test_number import SmsTestNumber





T = TypeVar("T", bound="SmsCampaign")



@_attrs_define
class SmsCampaign:
    """ 
        Attributes:
            access (SmsCampaignAccess):
            audience (list[SmsAudienceRef]):
            blocks (int):
            body (str):
            budget (SmsBudgetView):
            cost (float):
            costs (list[CountryLine]):
            counter (SmsCounter):
            created_by (str):
            funnel (SmsFunnel):
            id (str):
            links (list[SmsLinkView]):
            name (str):
            opt_out (str):
            prefix (str):
            price_providers (list[str]):
            retention_days (int):
            routes (list[SmsRouteLine]):
            schedule_mode (SmsCampaignScheduleMode):
            segments (int):
            short_domain (str):
            short_domain_status (SmsCampaignShortDomainStatus):
            status (SmsCampaignStatus):
            tests (list[SmsTestNumber]):
            text (str):
            texts (int):
            timezone (str):
            updated_at (datetime.datetime):
            updated_by (str):
            version (int):
            versions (int):
            warnings (int):
            window (str):
            autosaved_at (datetime.datetime | Unset):
            local_date (str | Unset):
            local_time (str | Unset):
            priced_on (datetime.datetime | Unset):
            send_at (datetime.datetime | Unset):
     """

    access: SmsCampaignAccess
    audience: list[SmsAudienceRef]
    blocks: int
    body: str
    budget: SmsBudgetView
    cost: float
    costs: list[CountryLine]
    counter: SmsCounter
    created_by: str
    funnel: SmsFunnel
    id: str
    links: list[SmsLinkView]
    name: str
    opt_out: str
    prefix: str
    price_providers: list[str]
    retention_days: int
    routes: list[SmsRouteLine]
    schedule_mode: SmsCampaignScheduleMode
    segments: int
    short_domain: str
    short_domain_status: SmsCampaignShortDomainStatus
    status: SmsCampaignStatus
    tests: list[SmsTestNumber]
    text: str
    texts: int
    timezone: str
    updated_at: datetime.datetime
    updated_by: str
    version: int
    versions: int
    warnings: int
    window: str
    autosaved_at: datetime.datetime | Unset = UNSET
    local_date: str | Unset = UNSET
    local_time: str | Unset = UNSET
    priced_on: datetime.datetime | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.country_line import CountryLine # noqa: PLC0415
        from ..models.sms_audience_ref import SmsAudienceRef # noqa: PLC0415
        from ..models.sms_budget_view import SmsBudgetView # noqa: PLC0415
        from ..models.sms_campaign_access import SmsCampaignAccess # noqa: PLC0415
        from ..models.sms_counter import SmsCounter # noqa: PLC0415
        from ..models.sms_funnel import SmsFunnel # noqa: PLC0415
        from ..models.sms_link_view import SmsLinkView # noqa: PLC0415
        from ..models.sms_route_line import SmsRouteLine # noqa: PLC0415
        from ..models.sms_test_number import SmsTestNumber # noqa: PLC0415
        access = self.access.to_dict()

        audience = []
        for audience_item_data in self.audience:
            audience_item = audience_item_data.to_dict()
            audience.append(audience_item)



        blocks = self.blocks

        body = self.body

        budget = self.budget.to_dict()

        cost = self.cost

        costs = []
        for costs_item_data in self.costs:
            costs_item = costs_item_data.to_dict()
            costs.append(costs_item)



        counter = self.counter.to_dict()

        created_by = self.created_by

        funnel = self.funnel.to_dict()

        id = self.id

        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        name = self.name

        opt_out = self.opt_out

        prefix = self.prefix

        price_providers = self.price_providers



        retention_days = self.retention_days

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        schedule_mode = self.schedule_mode.value

        segments = self.segments

        short_domain = self.short_domain

        short_domain_status = self.short_domain_status.value

        status = self.status.value

        tests = []
        for tests_item_data in self.tests:
            tests_item = tests_item_data.to_dict()
            tests.append(tests_item)



        text = self.text

        texts = self.texts

        timezone = self.timezone

        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        version = self.version

        versions = self.versions

        warnings = self.warnings

        window = self.window

        autosaved_at: str | Unset = UNSET
        if not isinstance(self.autosaved_at, Unset):
            autosaved_at = self.autosaved_at.isoformat()

        local_date = self.local_date

        local_time = self.local_time

        priced_on: str | Unset = UNSET
        if not isinstance(self.priced_on, Unset):
            priced_on = self.priced_on.isoformat()

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "audience": audience,
            "blocks": blocks,
            "body": body,
            "budget": budget,
            "cost": cost,
            "costs": costs,
            "counter": counter,
            "createdBy": created_by,
            "funnel": funnel,
            "id": id,
            "links": links,
            "name": name,
            "optOut": opt_out,
            "prefix": prefix,
            "priceProviders": price_providers,
            "retentionDays": retention_days,
            "routes": routes,
            "scheduleMode": schedule_mode,
            "segments": segments,
            "shortDomain": short_domain,
            "shortDomainStatus": short_domain_status,
            "status": status,
            "tests": tests,
            "text": text,
            "texts": texts,
            "timezone": timezone,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
            "version": version,
            "versions": versions,
            "warnings": warnings,
            "window": window,
        })
        if autosaved_at is not UNSET:
            field_dict["autosavedAt"] = autosaved_at
        if local_date is not UNSET:
            field_dict["localDate"] = local_date
        if local_time is not UNSET:
            field_dict["localTime"] = local_time
        if priced_on is not UNSET:
            field_dict["pricedOn"] = priced_on
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.country_line import CountryLine # noqa: PLC0415
        from ..models.sms_audience_ref import SmsAudienceRef # noqa: PLC0415
        from ..models.sms_budget_view import SmsBudgetView # noqa: PLC0415
        from ..models.sms_campaign_access import SmsCampaignAccess # noqa: PLC0415
        from ..models.sms_counter import SmsCounter # noqa: PLC0415
        from ..models.sms_funnel import SmsFunnel # noqa: PLC0415
        from ..models.sms_link_view import SmsLinkView # noqa: PLC0415
        from ..models.sms_route_line import SmsRouteLine # noqa: PLC0415
        from ..models.sms_test_number import SmsTestNumber # noqa: PLC0415
        d = dict(src_dict)
        access = SmsCampaignAccess.from_dict(d.pop("access"))




        audience = []
        _audience = d.pop("audience")
        for audience_item_data in (_audience):
            audience_item = SmsAudienceRef.from_dict(audience_item_data)



            audience.append(audience_item)


        blocks = d.pop("blocks")

        body = d.pop("body")

        budget = SmsBudgetView.from_dict(d.pop("budget"))




        cost = d.pop("cost")

        costs = []
        _costs = d.pop("costs")
        for costs_item_data in (_costs):
            costs_item = CountryLine.from_dict(costs_item_data)



            costs.append(costs_item)


        counter = SmsCounter.from_dict(d.pop("counter"))




        created_by = d.pop("createdBy")

        funnel = SmsFunnel.from_dict(d.pop("funnel"))




        id = d.pop("id")

        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = SmsLinkView.from_dict(links_item_data)



            links.append(links_item)


        name = d.pop("name")

        opt_out = d.pop("optOut")

        prefix = d.pop("prefix")

        price_providers = cast(list[str], d.pop("priceProviders"))


        retention_days = d.pop("retentionDays")

        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = SmsRouteLine.from_dict(routes_item_data)



            routes.append(routes_item)


        schedule_mode = SmsCampaignScheduleMode(d.pop("scheduleMode"))




        segments = d.pop("segments")

        short_domain = d.pop("shortDomain")

        short_domain_status = SmsCampaignShortDomainStatus(d.pop("shortDomainStatus"))




        status = SmsCampaignStatus(d.pop("status"))




        tests = []
        _tests = d.pop("tests")
        for tests_item_data in (_tests):
            tests_item = SmsTestNumber.from_dict(tests_item_data)



            tests.append(tests_item)


        text = d.pop("text")

        texts = d.pop("texts")

        timezone = d.pop("timezone")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        version = d.pop("version")

        versions = d.pop("versions")

        warnings = d.pop("warnings")

        window = d.pop("window")

        _autosaved_at = d.pop("autosavedAt", UNSET)
        autosaved_at: datetime.datetime | Unset
        if isinstance(_autosaved_at,  Unset):
            autosaved_at = UNSET
        else:
            autosaved_at = datetime.datetime.fromisoformat(_autosaved_at)




        local_date = d.pop("localDate", UNSET)

        local_time = d.pop("localTime", UNSET)

        _priced_on = d.pop("pricedOn", UNSET)
        priced_on: datetime.datetime | Unset
        if isinstance(_priced_on,  Unset):
            priced_on = UNSET
        else:
            priced_on = datetime.datetime.fromisoformat(_priced_on)




        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        sms_campaign = cls(
            access=access,
            audience=audience,
            blocks=blocks,
            body=body,
            budget=budget,
            cost=cost,
            costs=costs,
            counter=counter,
            created_by=created_by,
            funnel=funnel,
            id=id,
            links=links,
            name=name,
            opt_out=opt_out,
            prefix=prefix,
            price_providers=price_providers,
            retention_days=retention_days,
            routes=routes,
            schedule_mode=schedule_mode,
            segments=segments,
            short_domain=short_domain,
            short_domain_status=short_domain_status,
            status=status,
            tests=tests,
            text=text,
            texts=texts,
            timezone=timezone,
            updated_at=updated_at,
            updated_by=updated_by,
            version=version,
            versions=versions,
            warnings=warnings,
            window=window,
            autosaved_at=autosaved_at,
            local_date=local_date,
            local_time=local_time,
            priced_on=priced_on,
            send_at=send_at,
        )

        return sms_campaign

