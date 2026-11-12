from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="SmsTimePoint")



@_attrs_define
class SmsTimePoint:
    """ 
        Attributes:
            at (datetime.datetime):
            delivered (int):
     """

    at: datetime.datetime
    delivered: int





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        delivered = self.delivered


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "delivered": delivered,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        delivered = d.pop("delivered")

        sms_time_point = cls(
            at=at,
            delivered=delivered,
        )

        return sms_time_point

