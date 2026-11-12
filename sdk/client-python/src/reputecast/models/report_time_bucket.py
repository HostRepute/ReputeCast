from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportTimeBucket")



@_attrs_define
class ReportTimeBucket:
    """ 
        Attributes:
            clicks (int):
            human_opens (int):
            key (int):
     """

    clicks: int
    human_opens: int
    key: int





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        human_opens = self.human_opens

        key = self.key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "humanOpens": human_opens,
            "key": key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        human_opens = d.pop("humanOpens")

        key = d.pop("key")

        report_time_bucket = cls(
            clicks=clicks,
            human_opens=human_opens,
            key=key,
        )

        return report_time_bucket

