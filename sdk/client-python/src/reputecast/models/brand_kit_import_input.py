from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BrandKitImportInput")



@_attrs_define
class BrandKitImportInput:
    """ 
        Attributes:
            website (str): acme.io or https://acme.io
     """

    website: str





    def to_dict(self) -> dict[str, Any]:
        website = self.website


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "website": website,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        website = d.pop("website")

        brand_kit_import_input = cls(
            website=website,
        )

        return brand_kit_import_input

