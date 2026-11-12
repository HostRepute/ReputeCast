from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_copy_cost_pricing import PlacementCopyCostPricing
from ..types import UNSET, Unset






T = TypeVar("T", bound="PlacementCopyCost")



@_attrs_define
class PlacementCopyCost:
    """ 
        Attributes:
            copies (int):
            cost (float):
            pricing (PlacementCopyCostPricing):
            provider_id (str):
            short (str):
            plan_monthly (int | Unset):
            sent_today (int | Unset):
            today_cap (int | Unset):
            warm_day (int | Unset):
     """

    copies: int
    cost: float
    pricing: PlacementCopyCostPricing
    provider_id: str
    short: str
    plan_monthly: int | Unset = UNSET
    sent_today: int | Unset = UNSET
    today_cap: int | Unset = UNSET
    warm_day: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        copies = self.copies

        cost = self.cost

        pricing = self.pricing.value

        provider_id = self.provider_id

        short = self.short

        plan_monthly = self.plan_monthly

        sent_today = self.sent_today

        today_cap = self.today_cap

        warm_day = self.warm_day


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "copies": copies,
            "cost": cost,
            "pricing": pricing,
            "providerId": provider_id,
            "short": short,
        })
        if plan_monthly is not UNSET:
            field_dict["planMonthly"] = plan_monthly
        if sent_today is not UNSET:
            field_dict["sentToday"] = sent_today
        if today_cap is not UNSET:
            field_dict["todayCap"] = today_cap
        if warm_day is not UNSET:
            field_dict["warmDay"] = warm_day

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        copies = d.pop("copies")

        cost = d.pop("cost")

        pricing = PlacementCopyCostPricing(d.pop("pricing"))




        provider_id = d.pop("providerId")

        short = d.pop("short")

        plan_monthly = d.pop("planMonthly", UNSET)

        sent_today = d.pop("sentToday", UNSET)

        today_cap = d.pop("todayCap", UNSET)

        warm_day = d.pop("warmDay", UNSET)

        placement_copy_cost = cls(
            copies=copies,
            cost=cost,
            pricing=pricing,
            provider_id=provider_id,
            short=short,
            plan_monthly=plan_monthly,
            sent_today=sent_today,
            today_cap=today_cap,
            warm_day=warm_day,
        )

        return placement_copy_cost

