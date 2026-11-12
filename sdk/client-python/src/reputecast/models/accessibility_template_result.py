from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AccessibilityTemplateResult")



@_attrs_define
class AccessibilityTemplateResult:
    """ 
        Attributes:
            id (str):
            issues (int):
            serious (int):
            summary (str | Unset):
     """

    id: str
    issues: int
    serious: int
    summary: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        issues = self.issues

        serious = self.serious

        summary = self.summary


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "issues": issues,
            "serious": serious,
        })
        if summary is not UNSET:
            field_dict["summary"] = summary

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        issues = d.pop("issues")

        serious = d.pop("serious")

        summary = d.pop("summary", UNSET)

        accessibility_template_result = cls(
            id=id,
            issues=issues,
            serious=serious,
            summary=summary,
        )

        return accessibility_template_result

