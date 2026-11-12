from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_readiness_verdict import OverviewReadinessVerdict
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.overview_readiness_issue import OverviewReadinessIssue





T = TypeVar("T", bound="OverviewReadiness")



@_attrs_define
class OverviewReadiness:
    """ 
        Attributes:
            issues (list[OverviewReadinessIssue]):
            ran_at (datetime.datetime):
            verdict (OverviewReadinessVerdict):
     """

    issues: list[OverviewReadinessIssue]
    ran_at: datetime.datetime
    verdict: OverviewReadinessVerdict





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_readiness_issue import OverviewReadinessIssue # noqa: PLC0415
        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        ran_at = self.ran_at.isoformat()

        verdict = self.verdict.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "issues": issues,
            "ranAt": ran_at,
            "verdict": verdict,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_readiness_issue import OverviewReadinessIssue # noqa: PLC0415
        d = dict(src_dict)
        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = OverviewReadinessIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        ran_at = datetime.datetime.fromisoformat(d.pop("ranAt"))




        verdict = OverviewReadinessVerdict(d.pop("verdict"))




        overview_readiness = cls(
            issues=issues,
            ran_at=ran_at,
            verdict=verdict,
        )

        return overview_readiness

