from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_warmup_brief_status import ProviderWarmupBriefStatus






T = TypeVar("T", bound="ProviderWarmupBrief")



@_attrs_define
class ProviderWarmupBrief:
    """ 
        Attributes:
            day (int):
            days (int):
            plan_id (str):
            rate (float):
            sent_today (int):
            share (int):
            status (ProviderWarmupBriefStatus):
            today_cap (int):
            transactional_over_cap (int):
     """

    day: int
    days: int
    plan_id: str
    rate: float
    sent_today: int
    share: int
    status: ProviderWarmupBriefStatus
    today_cap: int
    transactional_over_cap: int





    def to_dict(self) -> dict[str, Any]:
        day = self.day

        days = self.days

        plan_id = self.plan_id

        rate = self.rate

        sent_today = self.sent_today

        share = self.share

        status = self.status.value

        today_cap = self.today_cap

        transactional_over_cap = self.transactional_over_cap


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "days": days,
            "planId": plan_id,
            "rate": rate,
            "sentToday": sent_today,
            "share": share,
            "status": status,
            "todayCap": today_cap,
            "transactionalOverCap": transactional_over_cap,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = d.pop("day")

        days = d.pop("days")

        plan_id = d.pop("planId")

        rate = d.pop("rate")

        sent_today = d.pop("sentToday")

        share = d.pop("share")

        status = ProviderWarmupBriefStatus(d.pop("status"))




        today_cap = d.pop("todayCap")

        transactional_over_cap = d.pop("transactionalOverCap")

        provider_warmup_brief = cls(
            day=day,
            days=days,
            plan_id=plan_id,
            rate=rate,
            sent_today=sent_today,
            share=share,
            status=status,
            today_cap=today_cap,
            transactional_over_cap=transactional_over_cap,
        )

        return provider_warmup_brief

