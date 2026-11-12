from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DeliverabilityWarmupProgress")



@_attrs_define
class DeliverabilityWarmupProgress:
    """ 
        Attributes:
            day (int):
            days (int):
     """

    day: int
    days: int





    def to_dict(self) -> dict[str, Any]:
        day = self.day

        days = self.days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "days": days,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = d.pop("day")

        days = d.pop("days")

        deliverability_warmup_progress = cls(
            day=day,
            days=days,
        )

        return deliverability_warmup_progress

