from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsTestNumberInput")



@_attrs_define
class SmsTestNumberInput:
    """ 
        Attributes:
            phone (str):
            user_id (str | Unset):
     """

    phone: str
    user_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        phone = self.phone

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "phone": phone,
        })
        if user_id is not UNSET:
            field_dict["userId"] = user_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phone = d.pop("phone")

        user_id = d.pop("userId", UNSET)

        sms_test_number_input = cls(
            phone=phone,
            user_id=user_id,
        )

        return sms_test_number_input

