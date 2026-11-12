from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsRouteTestInput")



@_attrs_define
class SmsRouteTestInput:
    """ 
        Attributes:
            phone (str):
            message (str | Unset):
     """

    phone: str
    message: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        phone = self.phone

        message = self.message


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "phone": phone,
        })
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phone = d.pop("phone")

        message = d.pop("message", UNSET)

        sms_route_test_input = cls(
            phone=phone,
            message=message,
        )

        return sms_route_test_input

