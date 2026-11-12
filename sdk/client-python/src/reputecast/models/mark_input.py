from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="MarkInput")



@_attrs_define
class MarkInput:
    """ 
        Attributes:
            blocks (list[str] | Unset):
     """

    blocks: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        blocks: list[str] | Unset = UNSET
        if not isinstance(self.blocks, Unset):
            blocks = self.blocks




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if blocks is not UNSET:
            field_dict["blocks"] = blocks

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocks = cast(list[str], d.pop("blocks", UNSET))


        mark_input = cls(
            blocks=blocks,
        )

        return mark_input

