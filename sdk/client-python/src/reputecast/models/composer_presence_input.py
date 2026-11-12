from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ComposerPresenceInput")



@_attrs_define
class ComposerPresenceInput:
    """ 
        Attributes:
            editing (bool | Unset):
     """

    editing: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        editing = self.editing


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if editing is not UNSET:
            field_dict["editing"] = editing

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        editing = d.pop("editing", UNSET)

        composer_presence_input = cls(
            editing=editing,
        )

        return composer_presence_input

