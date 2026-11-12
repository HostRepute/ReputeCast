from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SmsTestInput")



@_attrs_define
class SmsTestInput:
    """ 
        Attributes:
            numbers (list[str] | Unset):
     """

    numbers: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        numbers: list[str] | Unset = UNSET
        if not isinstance(self.numbers, Unset):
            numbers = self.numbers




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if numbers is not UNSET:
            field_dict["numbers"] = numbers

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        numbers = cast(list[str], d.pop("numbers", UNSET))


        sms_test_input = cls(
            numbers=numbers,
        )

        return sms_test_input

