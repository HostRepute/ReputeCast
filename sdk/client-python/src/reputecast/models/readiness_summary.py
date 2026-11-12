from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_summary_verdict import ReadinessSummaryVerdict






T = TypeVar("T", bound="ReadinessSummary")



@_attrs_define
class ReadinessSummary:
    """ 
        Attributes:
            issues (int):
            verdict (ReadinessSummaryVerdict):
     """

    issues: int
    verdict: ReadinessSummaryVerdict





    def to_dict(self) -> dict[str, Any]:
        issues = self.issues

        verdict = self.verdict.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "issues": issues,
            "verdict": verdict,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        issues = d.pop("issues")

        verdict = ReadinessSummaryVerdict(d.pop("verdict"))




        readiness_summary = cls(
            issues=issues,
            verdict=verdict,
        )

        return readiness_summary

