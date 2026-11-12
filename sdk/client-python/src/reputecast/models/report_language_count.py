from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportLanguageCount")



@_attrs_define
class ReportLanguageCount:
    """ 
        Attributes:
            count (int):
            language (str):
            name (str):
     """

    count: int
    language: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        language = self.language

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "language": language,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        language = d.pop("language")

        name = d.pop("name")

        report_language_count = cls(
            count=count,
            language=language,
            name=name,
        )

        return report_language_count

