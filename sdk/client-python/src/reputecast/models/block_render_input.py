from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="BlockRenderInput")



@_attrs_define
class BlockRenderInput:
    """ 
        Attributes:
            settings (Any | Unset):
     """

    settings: Any | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        settings = self.settings


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if settings is not UNSET:
            field_dict["settings"] = settings

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        settings = d.pop("settings", UNSET)

        block_render_input = cls(
            settings=settings,
        )

        return block_render_input

