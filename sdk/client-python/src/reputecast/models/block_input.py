from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="BlockInput")



@_attrs_define
class BlockInput:
    """ 
        Attributes:
            index (int | Unset):
            key (str | Unset):
     """

    index: int | Unset = UNSET
    key: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        index = self.index

        key = self.key


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if index is not UNSET:
            field_dict["index"] = index
        if key is not UNSET:
            field_dict["key"] = key

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        index = d.pop("index", UNSET)

        key = d.pop("key", UNSET)

        block_input = cls(
            index=index,
            key=key,
        )

        return block_input

