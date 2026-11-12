from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_block import CommerceBlock





T = TypeVar("T", bound="CommerceBlockRenderInput")



@_attrs_define
class CommerceBlockRenderInput:
    """ 
        Attributes:
            block (CommerceBlock):
     """

    block: CommerceBlock





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_block import CommerceBlock # noqa: PLC0415
        block = self.block.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "block": block,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_block import CommerceBlock # noqa: PLC0415
        d = dict(src_dict)
        block = CommerceBlock.from_dict(d.pop("block"))




        commerce_block_render_input = cls(
            block=block,
        )

        return commerce_block_render_input

