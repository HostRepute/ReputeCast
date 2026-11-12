from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SmsSuppressionAdd")



@_attrs_define
class SmsSuppressionAdd:
    """ 
        Attributes:
            numbers (list[str]):
            note (str | Unset):
     """

    numbers: list[str]
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        numbers = self.numbers



        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "numbers": numbers,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        numbers = cast(list[str], d.pop("numbers"))


        note = d.pop("note", UNSET)

        sms_suppression_add = cls(
            numbers=numbers,
            note=note,
        )

        return sms_suppression_add

