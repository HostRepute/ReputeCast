from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushReportPoint")



@_attrs_define
class PushReportPoint:
    """ 
        Attributes:
            clicks (int):
            minutes (int):
     """

    clicks: int
    minutes: int





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        minutes = self.minutes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "minutes": minutes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        minutes = d.pop("minutes")

        push_report_point = cls(
            clicks=clicks,
            minutes=minutes,
        )

        return push_report_point

