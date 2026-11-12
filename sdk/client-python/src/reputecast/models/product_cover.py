from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductCover")



@_attrs_define
class ProductCover:
    """ 
        Attributes:
            height (int):
            url (str):
            width (int):
     """

    height: int
    url: str
    width: int





    def to_dict(self) -> dict[str, Any]:
        height = self.height

        url = self.url

        width = self.width


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "height": height,
            "url": url,
            "width": width,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        height = d.pop("height")

        url = d.pop("url")

        width = d.pop("width")

        product_cover = cls(
            height=height,
            url=url,
            width=width,
        )

        return product_cover

