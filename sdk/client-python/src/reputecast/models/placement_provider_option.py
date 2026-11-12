from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_provider_option_pricing import PlacementProviderOptionPricing
from ..models.placement_provider_option_status import PlacementProviderOptionStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PlacementProviderOption")



@_attrs_define
class PlacementProviderOption:
    """ 
        Attributes:
            adapter (str):
            available (bool):
            color (str):
            cost_per_copy (float):
            detail (str): us-east-1, mta.acme.io, the Transactional stream
            id (str):
            name (str):
            pricing (PlacementProviderOptionPricing):
            short (str):
            status (PlacementProviderOptionStatus):
            down_reason (str | Unset):
            down_since (datetime.datetime | Unset):
            plan_monthly (int | Unset):
            sent_today (int | Unset):
            stream_only (str | Unset):
            today_cap (int | Unset):
            warm_day (int | Unset):
            warm_days (int | Unset):
     """

    adapter: str
    available: bool
    color: str
    cost_per_copy: float
    detail: str
    id: str
    name: str
    pricing: PlacementProviderOptionPricing
    short: str
    status: PlacementProviderOptionStatus
    down_reason: str | Unset = UNSET
    down_since: datetime.datetime | Unset = UNSET
    plan_monthly: int | Unset = UNSET
    sent_today: int | Unset = UNSET
    stream_only: str | Unset = UNSET
    today_cap: int | Unset = UNSET
    warm_day: int | Unset = UNSET
    warm_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        available = self.available

        color = self.color

        cost_per_copy = self.cost_per_copy

        detail = self.detail

        id = self.id

        name = self.name

        pricing = self.pricing.value

        short = self.short

        status = self.status.value

        down_reason = self.down_reason

        down_since: str | Unset = UNSET
        if not isinstance(self.down_since, Unset):
            down_since = self.down_since.isoformat()

        plan_monthly = self.plan_monthly

        sent_today = self.sent_today

        stream_only = self.stream_only

        today_cap = self.today_cap

        warm_day = self.warm_day

        warm_days = self.warm_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "available": available,
            "color": color,
            "costPerCopy": cost_per_copy,
            "detail": detail,
            "id": id,
            "name": name,
            "pricing": pricing,
            "short": short,
            "status": status,
        })
        if down_reason is not UNSET:
            field_dict["downReason"] = down_reason
        if down_since is not UNSET:
            field_dict["downSince"] = down_since
        if plan_monthly is not UNSET:
            field_dict["planMonthly"] = plan_monthly
        if sent_today is not UNSET:
            field_dict["sentToday"] = sent_today
        if stream_only is not UNSET:
            field_dict["streamOnly"] = stream_only
        if today_cap is not UNSET:
            field_dict["todayCap"] = today_cap
        if warm_day is not UNSET:
            field_dict["warmDay"] = warm_day
        if warm_days is not UNSET:
            field_dict["warmDays"] = warm_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        available = d.pop("available")

        color = d.pop("color")

        cost_per_copy = d.pop("costPerCopy")

        detail = d.pop("detail")

        id = d.pop("id")

        name = d.pop("name")

        pricing = PlacementProviderOptionPricing(d.pop("pricing"))




        short = d.pop("short")

        status = PlacementProviderOptionStatus(d.pop("status"))




        down_reason = d.pop("downReason", UNSET)

        _down_since = d.pop("downSince", UNSET)
        down_since: datetime.datetime | Unset
        if isinstance(_down_since,  Unset):
            down_since = UNSET
        else:
            down_since = datetime.datetime.fromisoformat(_down_since)




        plan_monthly = d.pop("planMonthly", UNSET)

        sent_today = d.pop("sentToday", UNSET)

        stream_only = d.pop("streamOnly", UNSET)

        today_cap = d.pop("todayCap", UNSET)

        warm_day = d.pop("warmDay", UNSET)

        warm_days = d.pop("warmDays", UNSET)

        placement_provider_option = cls(
            adapter=adapter,
            available=available,
            color=color,
            cost_per_copy=cost_per_copy,
            detail=detail,
            id=id,
            name=name,
            pricing=pricing,
            short=short,
            status=status,
            down_reason=down_reason,
            down_since=down_since,
            plan_monthly=plan_monthly,
            sent_today=sent_today,
            stream_only=stream_only,
            today_cap=today_cap,
            warm_day=warm_day,
            warm_days=warm_days,
        )

        return placement_provider_option

