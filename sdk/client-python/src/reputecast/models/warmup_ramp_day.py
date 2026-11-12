from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WarmupRampDay")



@_attrs_define
class WarmupRampDay:
    """ 
        Attributes:
            day (int):
            planned_cap (int):
            slowed (bool):
            today (bool):
            sent (int | Unset):
     """

    day: int
    planned_cap: int
    slowed: bool
    today: bool
    sent: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        day = self.day

        planned_cap = self.planned_cap

        slowed = self.slowed

        today = self.today

        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "plannedCap": planned_cap,
            "slowed": slowed,
            "today": today,
        })
        if sent is not UNSET:
            field_dict["sent"] = sent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = d.pop("day")

        planned_cap = d.pop("plannedCap")

        slowed = d.pop("slowed")

        today = d.pop("today")

        sent = d.pop("sent", UNSET)

        warmup_ramp_day = cls(
            day=day,
            planned_cap=planned_cap,
            slowed=slowed,
            today=today,
            sent=sent,
        )

        return warmup_ramp_day

