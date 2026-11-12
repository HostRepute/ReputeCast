from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_readiness_view_verdict import ApprovalReadinessViewVerdict
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.readiness_check import ReadinessCheck





T = TypeVar("T", bound="ApprovalReadinessView")



@_attrs_define
class ApprovalReadinessView:
    """ 
        Attributes:
            blocks (int):
            checked_at (datetime.datetime):
            issues (list[ReadinessCheck]):
            passed (int):
            total (int):
            verdict (ApprovalReadinessViewVerdict):
            warnings (int):
     """

    blocks: int
    checked_at: datetime.datetime
    issues: list[ReadinessCheck]
    passed: int
    total: int
    verdict: ApprovalReadinessViewVerdict
    warnings: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        blocks = self.blocks

        checked_at = self.checked_at.isoformat()

        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        passed = self.passed

        total = self.total

        verdict = self.verdict.value

        warnings = self.warnings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "checkedAt": checked_at,
            "issues": issues,
            "passed": passed,
            "total": total,
            "verdict": verdict,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        d = dict(src_dict)
        blocks = d.pop("blocks")

        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = ReadinessCheck.from_dict(issues_item_data)



            issues.append(issues_item)


        passed = d.pop("passed")

        total = d.pop("total")

        verdict = ApprovalReadinessViewVerdict(d.pop("verdict"))




        warnings = d.pop("warnings")

        approval_readiness_view = cls(
            blocks=blocks,
            checked_at=checked_at,
            issues=issues,
            passed=passed,
            total=total,
            verdict=verdict,
            warnings=warnings,
        )

        return approval_readiness_view

