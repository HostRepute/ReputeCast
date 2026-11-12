from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ImportLanguageValue")



@_attrs_define
class ImportLanguageValue:
    """ 
        Attributes:
            code (str):
            name (str):
            rows (int):
            value (str):
     """

    code: str
    name: str
    rows: int
    value: str





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        name = self.name

        rows = self.rows

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "name": name,
            "rows": rows,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        name = d.pop("name")

        rows = d.pop("rows")

        value = d.pop("value")

        import_language_value = cls(
            code=code,
            name=name,
            rows=rows,
            value=value,
        )

        return import_language_value

