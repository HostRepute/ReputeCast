from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReadinessCounts")



@_attrs_define
class ReadinessCounts:
    """ 
        Attributes:
            blocks (int):
            passed (int):
            total (int):
            unknown (int):
            warnings (int):
     """

    blocks: int
    passed: int
    total: int
    unknown: int
    warnings: int





    def to_dict(self) -> dict[str, Any]:
        blocks = self.blocks

        passed = self.passed

        total = self.total

        unknown = self.unknown

        warnings = self.warnings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "passed": passed,
            "total": total,
            "unknown": unknown,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocks = d.pop("blocks")

        passed = d.pop("passed")

        total = d.pop("total")

        unknown = d.pop("unknown")

        warnings = d.pop("warnings")

        readiness_counts = cls(
            blocks=blocks,
            passed=passed,
            total=total,
            unknown=unknown,
            warnings=warnings,
        )

        return readiness_counts

