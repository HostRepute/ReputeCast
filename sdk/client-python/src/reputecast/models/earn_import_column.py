from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_import_column_used_as import EarnImportColumnUsedAs






T = TypeVar("T", bound="EarnImportColumn")



@_attrs_define
class EarnImportColumn:
    """ 
        Attributes:
            example (str):
            name (str):
            used_as (EarnImportColumnUsedAs):
     """

    example: str
    name: str
    used_as: EarnImportColumnUsedAs





    def to_dict(self) -> dict[str, Any]:
        example = self.example

        name = self.name

        used_as = self.used_as.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "example": example,
            "name": name,
            "usedAs": used_as,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        example = d.pop("example")

        name = d.pop("name")

        used_as = EarnImportColumnUsedAs(d.pop("usedAs"))




        earn_import_column = cls(
            example=example,
            name=name,
            used_as=used_as,
        )

        return earn_import_column

