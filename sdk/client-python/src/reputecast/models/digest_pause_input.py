from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="DigestPauseInput")



@_attrs_define
class DigestPauseInput:
    """ 
        Attributes:
            note (str | Unset):
     """

    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        note = d.pop("note", UNSET)

        digest_pause_input = cls(
            note=note,
        )

        return digest_pause_input

