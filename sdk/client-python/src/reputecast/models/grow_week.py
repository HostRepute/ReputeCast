from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="GrowWeek")



@_attrs_define
class GrowWeek:
    """ 
        Attributes:
            days (int):
            end (datetime.datetime):
            other (int): Forms and other sources (imports left out)
            partial (bool): The week is still going
            partners (int):
            referrals (int):
            start (datetime.datetime):
     """

    days: int
    end: datetime.datetime
    other: int
    partial: bool
    partners: int
    referrals: int
    start: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        days = self.days

        end = self.end.isoformat()

        other = self.other

        partial = self.partial

        partners = self.partners

        referrals = self.referrals

        start = self.start.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "end": end,
            "other": other,
            "partial": partial,
            "partners": partners,
            "referrals": referrals,
            "start": start,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        days = d.pop("days")

        end = datetime.datetime.fromisoformat(d.pop("end"))




        other = d.pop("other")

        partial = d.pop("partial")

        partners = d.pop("partners")

        referrals = d.pop("referrals")

        start = datetime.datetime.fromisoformat(d.pop("start"))




        grow_week = cls(
            days=days,
            end=end,
            other=other,
            partial=partial,
            partners=partners,
            referrals=referrals,
            start=start,
        )

        return grow_week

