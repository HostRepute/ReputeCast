from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReferralBucket")



@_attrs_define
class ReferralBucket:
    """ 
        Attributes:
            confirmed (int):
            days (int):
            end (datetime.datetime):
            partial (bool):
            start (datetime.datetime):
     """

    confirmed: int
    days: int
    end: datetime.datetime
    partial: bool
    start: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        confirmed = self.confirmed

        days = self.days

        end = self.end.isoformat()

        partial = self.partial

        start = self.start.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmed": confirmed,
            "days": days,
            "end": end,
            "partial": partial,
            "start": start,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confirmed = d.pop("confirmed")

        days = d.pop("days")

        end = datetime.datetime.fromisoformat(d.pop("end"))




        partial = d.pop("partial")

        start = datetime.datetime.fromisoformat(d.pop("start"))




        referral_bucket = cls(
            confirmed=confirmed,
            days=days,
            end=end,
            partial=partial,
            start=start,
        )

        return referral_bucket

