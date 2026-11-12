from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityPostmasterPeak")



@_attrs_define
class DeliverabilityPostmasterPeak:
    """ 
        Attributes:
            day (datetime.date):
            spam_rate (float):
            note (str | Unset):
     """

    day: datetime.date
    spam_rate: float
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        day = self.day.isoformat()

        spam_rate = self.spam_rate

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "spamRate": spam_rate,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = datetime.date.fromisoformat(d.pop("day"))




        spam_rate = d.pop("spamRate")

        note = d.pop("note", UNSET)

        deliverability_postmaster_peak = cls(
            day=day,
            spam_rate=spam_rate,
            note=note,
        )

        return deliverability_postmaster_peak

