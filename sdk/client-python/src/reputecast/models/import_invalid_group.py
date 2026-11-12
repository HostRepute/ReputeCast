from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ImportInvalidGroup")



@_attrs_define
class ImportInvalidGroup:
    """ 
        Attributes:
            problem (str):
            rows (int):
            example (str | Unset):
            service (str | Unset):
     """

    problem: str
    rows: int
    example: str | Unset = UNSET
    service: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        problem = self.problem

        rows = self.rows

        example = self.example

        service = self.service


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "problem": problem,
            "rows": rows,
        })
        if example is not UNSET:
            field_dict["example"] = example
        if service is not UNSET:
            field_dict["service"] = service

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        problem = d.pop("problem")

        rows = d.pop("rows")

        example = d.pop("example", UNSET)

        service = d.pop("service", UNSET)

        import_invalid_group = cls(
            problem=problem,
            rows=rows,
            example=example,
            service=service,
        )

        return import_invalid_group

