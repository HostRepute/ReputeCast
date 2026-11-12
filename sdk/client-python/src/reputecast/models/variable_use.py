from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.variable_use_where import VariableUseWhere






T = TypeVar("T", bound="VariableUse")



@_attrs_define
class VariableUse:
    """ 
        Attributes:
            column (int):
            line (int):
            name (str):
            where (VariableUseWhere):
     """

    column: int
    line: int
    name: str
    where: VariableUseWhere





    def to_dict(self) -> dict[str, Any]:
        column = self.column

        line = self.line

        name = self.name

        where = self.where.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "column": column,
            "line": line,
            "name": name,
            "where": where,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        column = d.pop("column")

        line = d.pop("line")

        name = d.pop("name")

        where = VariableUseWhere(d.pop("where"))




        variable_use = cls(
            column=column,
            line=line,
            name=name,
            where=where,
        )

        return variable_use

