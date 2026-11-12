from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PlacementLanguage")



@_attrs_define
class PlacementLanguage:
    """ 
        Attributes:
            code (str):
            source (bool):
     """

    code: str
    source: bool





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        source = d.pop("source")

        placement_language = cls(
            code=code,
            source=source,
        )

        return placement_language

