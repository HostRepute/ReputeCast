from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SourceInput")



@_attrs_define
class SourceInput:
    """ 
        Attributes:
            language (str):
     """

    language: str





    def to_dict(self) -> dict[str, Any]:
        language = self.language


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "language": language,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        language = d.pop("language")

        source_input = cls(
            language=language,
        )

        return source_input

