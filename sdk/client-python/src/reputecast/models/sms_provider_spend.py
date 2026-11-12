from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsProviderSpend")



@_attrs_define
class SmsProviderSpend:
    """ 
        Attributes:
            name (str):
            spend (float):
     """

    name: str
    spend: float





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        spend = self.spend


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "spend": spend,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        spend = d.pop("spend")

        sms_provider_spend = cls(
            name=name,
            spend=spend,
        )

        return sms_provider_spend

