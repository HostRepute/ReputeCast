from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="KeepUpToDateInput")



@_attrs_define
class KeepUpToDateInput:
    """ 
        Attributes:
            blocks (list[str]):
     """

    blocks: list[str]





    def to_dict(self) -> dict[str, Any]:
        blocks = self.blocks




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocks = cast(list[str], d.pop("blocks"))


        keep_up_to_date_input = cls(
            blocks=blocks,
        )

        return keep_up_to_date_input

