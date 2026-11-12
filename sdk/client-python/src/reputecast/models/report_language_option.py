from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportLanguageOption")



@_attrs_define
class ReportLanguageOption:
    """ 
        Attributes:
            language (str):
            name (str):
     """

    language: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        language = self.language

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "language": language,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        language = d.pop("language")

        name = d.pop("name")

        report_language_option = cls(
            language=language,
            name=name,
        )

        return report_language_option

