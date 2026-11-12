from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsCountryReport")



@_attrs_define
class SmsCountryReport:
    """ 
        Attributes:
            click_pct (float):
            clicks (int):
            cost (float):
            country (str):
            delivered (int):
            delivered_pct (float):
            failed (int):
            opt_outs (int):
            sender (str):
            sent (int):
            skipped (int):
            skip_reason (str | Unset):
     """

    click_pct: float
    clicks: int
    cost: float
    country: str
    delivered: int
    delivered_pct: float
    failed: int
    opt_outs: int
    sender: str
    sent: int
    skipped: int
    skip_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        click_pct = self.click_pct

        clicks = self.clicks

        cost = self.cost

        country = self.country

        delivered = self.delivered

        delivered_pct = self.delivered_pct

        failed = self.failed

        opt_outs = self.opt_outs

        sender = self.sender

        sent = self.sent

        skipped = self.skipped

        skip_reason = self.skip_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clickPct": click_pct,
            "clicks": clicks,
            "cost": cost,
            "country": country,
            "delivered": delivered,
            "deliveredPct": delivered_pct,
            "failed": failed,
            "optOuts": opt_outs,
            "sender": sender,
            "sent": sent,
            "skipped": skipped,
        })
        if skip_reason is not UNSET:
            field_dict["skipReason"] = skip_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        click_pct = d.pop("clickPct")

        clicks = d.pop("clicks")

        cost = d.pop("cost")

        country = d.pop("country")

        delivered = d.pop("delivered")

        delivered_pct = d.pop("deliveredPct")

        failed = d.pop("failed")

        opt_outs = d.pop("optOuts")

        sender = d.pop("sender")

        sent = d.pop("sent")

        skipped = d.pop("skipped")

        skip_reason = d.pop("skipReason", UNSET)

        sms_country_report = cls(
            click_pct=click_pct,
            clicks=clicks,
            cost=cost,
            country=country,
            delivered=delivered,
            delivered_pct=delivered_pct,
            failed=failed,
            opt_outs=opt_outs,
            sender=sender,
            sent=sent,
            skipped=skipped,
            skip_reason=skip_reason,
        )

        return sms_country_report

