from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PushMonth")



@_attrs_define
class PushMonth:
    """ 
        Attributes:
            campaigns (int):
            month (datetime.datetime):
            sent (int):
            shown_pct (float):
     """

    campaigns: int
    month: datetime.datetime
    sent: int
    shown_pct: float





    def to_dict(self) -> dict[str, Any]:
        campaigns = self.campaigns

        month = self.month.isoformat()

        sent = self.sent

        shown_pct = self.shown_pct


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaigns": campaigns,
            "month": month,
            "sent": sent,
            "shownPct": shown_pct,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaigns = d.pop("campaigns")

        month = datetime.datetime.fromisoformat(d.pop("month"))




        sent = d.pop("sent")

        shown_pct = d.pop("shownPct")

        push_month = cls(
            campaigns=campaigns,
            month=month,
            sent=sent,
            shown_pct=shown_pct,
        )

        return push_month

