from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="AddSuppressionsInput")



@_attrs_define
class AddSuppressionsInput:
    """ 
        Attributes:
            emails (list[str]):
            note (str | Unset): Why, shown as the original code
     """

    emails: list[str]
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        emails = self.emails



        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "emails": emails,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        emails = cast(list[str], d.pop("emails"))


        note = d.pop("note", UNSET)

        add_suppressions_input = cls(
            emails=emails,
            note=note,
        )

        return add_suppressions_input

