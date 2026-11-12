from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.brand_button_style_shape import BrandButtonStyleShape
from ..models.brand_button_style_size import BrandButtonStyleSize






T = TypeVar("T", bound="BrandButtonStyle")



@_attrs_define
class BrandButtonStyle:
    """ 
        Attributes:
            shape (BrandButtonStyleShape):
            size (BrandButtonStyleSize): Text size in px (bold, body font)
     """

    shape: BrandButtonStyleShape
    size: BrandButtonStyleSize





    def to_dict(self) -> dict[str, Any]:
        shape = self.shape.value

        size = self.size.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "shape": shape,
            "size": size,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        shape = BrandButtonStyleShape(d.pop("shape"))




        size = BrandButtonStyleSize(d.pop("size"))




        brand_button_style = cls(
            shape=shape,
            size=size,
        )

        return brand_button_style

