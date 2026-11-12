from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationLanguageMark")



@_attrs_define
class AutomationLanguageMark:
    """ 
        Attributes:
            version (int):
     """

    version: int





    def to_dict(self) -> dict[str, Any]:
        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        automation_language_mark = cls(
            version=version,
        )

        return automation_language_mark

