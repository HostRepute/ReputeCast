from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_route_share_color import ReadinessRouteShareColor
from ..models.readiness_route_share_pricing import ReadinessRouteSharePricing
from ..models.readiness_route_share_rate_from import ReadinessRouteShareRateFrom
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReadinessRouteShare")



@_attrs_define
class ReadinessRouteShare:
    """ 
        Attributes:
            adapter (str):
            available (bool):
            color (ReadinessRouteShareColor):
            cost (float):
            minutes (int):
            name (str):
            percent (float):
            price_label (str):
            pricing (ReadinessRouteSharePricing):
            provider_id (str):
            rate (float):
            rate_label (str):
            recipients (int):
            short (str):
            status (str):
            status_label (str):
            days (int | Unset):
            label (str | Unset):
            over_quota (int | Unset):
            rate_from (ReadinessRouteShareRateFrom | Unset):
            warmup_cap (int | Unset):
            warmup_day (int | Unset):
            warmup_days (int | Unset):
     """

    adapter: str
    available: bool
    color: ReadinessRouteShareColor
    cost: float
    minutes: int
    name: str
    percent: float
    price_label: str
    pricing: ReadinessRouteSharePricing
    provider_id: str
    rate: float
    rate_label: str
    recipients: int
    short: str
    status: str
    status_label: str
    days: int | Unset = UNSET
    label: str | Unset = UNSET
    over_quota: int | Unset = UNSET
    rate_from: ReadinessRouteShareRateFrom | Unset = UNSET
    warmup_cap: int | Unset = UNSET
    warmup_day: int | Unset = UNSET
    warmup_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        available = self.available

        color = self.color.value

        cost = self.cost

        minutes = self.minutes

        name = self.name

        percent = self.percent

        price_label = self.price_label

        pricing = self.pricing.value

        provider_id = self.provider_id

        rate = self.rate

        rate_label = self.rate_label

        recipients = self.recipients

        short = self.short

        status = self.status

        status_label = self.status_label

        days = self.days

        label = self.label

        over_quota = self.over_quota

        rate_from: str | Unset = UNSET
        if not isinstance(self.rate_from, Unset):
            rate_from = self.rate_from.value


        warmup_cap = self.warmup_cap

        warmup_day = self.warmup_day

        warmup_days = self.warmup_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "available": available,
            "color": color,
            "cost": cost,
            "minutes": minutes,
            "name": name,
            "percent": percent,
            "priceLabel": price_label,
            "pricing": pricing,
            "providerId": provider_id,
            "rate": rate,
            "rateLabel": rate_label,
            "recipients": recipients,
            "short": short,
            "status": status,
            "statusLabel": status_label,
        })
        if days is not UNSET:
            field_dict["days"] = days
        if label is not UNSET:
            field_dict["label"] = label
        if over_quota is not UNSET:
            field_dict["overQuota"] = over_quota
        if rate_from is not UNSET:
            field_dict["rateFrom"] = rate_from
        if warmup_cap is not UNSET:
            field_dict["warmupCap"] = warmup_cap
        if warmup_day is not UNSET:
            field_dict["warmupDay"] = warmup_day
        if warmup_days is not UNSET:
            field_dict["warmupDays"] = warmup_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        available = d.pop("available")

        color = ReadinessRouteShareColor(d.pop("color"))




        cost = d.pop("cost")

        minutes = d.pop("minutes")

        name = d.pop("name")

        percent = d.pop("percent")

        price_label = d.pop("priceLabel")

        pricing = ReadinessRouteSharePricing(d.pop("pricing"))




        provider_id = d.pop("providerId")

        rate = d.pop("rate")

        rate_label = d.pop("rateLabel")

        recipients = d.pop("recipients")

        short = d.pop("short")

        status = d.pop("status")

        status_label = d.pop("statusLabel")

        days = d.pop("days", UNSET)

        label = d.pop("label", UNSET)

        over_quota = d.pop("overQuota", UNSET)

        _rate_from = d.pop("rateFrom", UNSET)
        rate_from: ReadinessRouteShareRateFrom | Unset
        if isinstance(_rate_from,  Unset):
            rate_from = UNSET
        else:
            rate_from = ReadinessRouteShareRateFrom(_rate_from)




        warmup_cap = d.pop("warmupCap", UNSET)

        warmup_day = d.pop("warmupDay", UNSET)

        warmup_days = d.pop("warmupDays", UNSET)

        readiness_route_share = cls(
            adapter=adapter,
            available=available,
            color=color,
            cost=cost,
            minutes=minutes,
            name=name,
            percent=percent,
            price_label=price_label,
            pricing=pricing,
            provider_id=provider_id,
            rate=rate,
            rate_label=rate_label,
            recipients=recipients,
            short=short,
            status=status,
            status_label=status_label,
            days=days,
            label=label,
            over_quota=over_quota,
            rate_from=rate_from,
            warmup_cap=warmup_cap,
            warmup_day=warmup_day,
            warmup_days=warmup_days,
        )

        return readiness_route_share

