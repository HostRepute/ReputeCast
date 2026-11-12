from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HoldInput")



@_attrs_define
class HoldInput:
    """ 
        Attributes:
            held (bool):
     """

    held: bool





    def to_dict(self) -> dict[str, Any]:
        held = self.held


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "held": held,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        held = d.pop("held")

        hold_input = cls(
            held=held,
        )

        return hold_input

