from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteMeterUse")



@_attrs_define
class WebsiteMeterUse:
    """ 
        Attributes:
            limit (int):
            month (datetime.datetime):
            used (int):
     """

    limit: int
    month: datetime.datetime
    used: int





    def to_dict(self) -> dict[str, Any]:
        limit = self.limit

        month = self.month.isoformat()

        used = self.used


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "limit": limit,
            "month": month,
            "used": used,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        limit = d.pop("limit")

        month = datetime.datetime.fromisoformat(d.pop("month"))




        used = d.pop("used")

        website_meter_use = cls(
            limit=limit,
            month=month,
            used=used,
        )

        return website_meter_use

