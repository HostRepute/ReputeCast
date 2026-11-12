from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AccessibilityLanguage")



@_attrs_define
class AccessibilityLanguage:
    """ 
        Attributes:
            code (str):  Example: es.
            name (str):  Example: Spanish.
            source (bool):
     """

    code: str
    name: str
    source: bool





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        name = self.name

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "name": name,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        name = d.pop("name")

        source = d.pop("source")

        accessibility_language = cls(
            code=code,
            name=name,
            source=source,
        )

        return accessibility_language

