from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsPhoneInput")



@_attrs_define
class SmsPhoneInput:
    """ 
        Attributes:
            phone (str):
            country (str | Unset):
     """

    phone: str
    country: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        phone = self.phone

        country = self.country


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "phone": phone,
        })
        if country is not UNSET:
            field_dict["country"] = country

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phone = d.pop("phone")

        country = d.pop("country", UNSET)

        sms_phone_input = cls(
            phone=phone,
            country=country,
        )

        return sms_phone_input

