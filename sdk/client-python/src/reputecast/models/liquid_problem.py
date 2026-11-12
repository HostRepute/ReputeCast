from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiquidProblem")



@_attrs_define
class LiquidProblem:
    """ 
        Attributes:
            line (int):
            message (str):
     """

    line: int
    message: str





    def to_dict(self) -> dict[str, Any]:
        line = self.line

        message = self.message


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "line": line,
            "message": message,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        line = d.pop("line")

        message = d.pop("message")

        liquid_problem = cls(
            line=line,
            message=message,
        )

        return liquid_problem

