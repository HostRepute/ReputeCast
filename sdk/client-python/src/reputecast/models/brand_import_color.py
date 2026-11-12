from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BrandImportColor")



@_attrs_define
class BrandImportColor:
    """ 
        Attributes:
            hex_ (str):
            used (bool):
     """

    hex_: str
    used: bool





    def to_dict(self) -> dict[str, Any]:
        hex_ = self.hex_

        used = self.used


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hex": hex_,
            "used": used,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hex_ = d.pop("hex")

        used = d.pop("used")

        brand_import_color = cls(
            hex_=hex_,
            used=used,
        )

        return brand_import_color

