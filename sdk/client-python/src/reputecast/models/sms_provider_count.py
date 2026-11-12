from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsProviderCount")



@_attrs_define
class SmsProviderCount:
    """ 
        Attributes:
            count (int):
            name (str):
     """

    count: int
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        name = d.pop("name")

        sms_provider_count = cls(
            count=count,
            name=name,
        )

        return sms_provider_count

