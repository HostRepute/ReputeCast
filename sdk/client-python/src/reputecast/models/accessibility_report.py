from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_report_mode import AccessibilityReportMode
from ..models.accessibility_report_policy import AccessibilityReportPolicy
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.accessibility_counts import AccessibilityCounts
  from ..models.accessibility_issue import AccessibilityIssue
  from ..models.accessibility_language import AccessibilityLanguage





T = TypeVar("T", bound="AccessibilityReport")



@_attrs_define
class AccessibilityReport:
    """ 
        Attributes:
            checked_at (datetime.datetime):
            counts (AccessibilityCounts):
            duration_ms (int):
            issues (list[AccessibilityIssue]):
            languages (list[AccessibilityLanguage]):
            min_text_px (int):
            mode (AccessibilityReportMode):
            moderate (int):
            policy (AccessibilityReportPolicy):
            serious (int):
     """

    checked_at: datetime.datetime
    counts: AccessibilityCounts
    duration_ms: int
    issues: list[AccessibilityIssue]
    languages: list[AccessibilityLanguage]
    min_text_px: int
    mode: AccessibilityReportMode
    moderate: int
    policy: AccessibilityReportPolicy
    serious: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.accessibility_counts import AccessibilityCounts # noqa: PLC0415
        from ..models.accessibility_issue import AccessibilityIssue # noqa: PLC0415
        from ..models.accessibility_language import AccessibilityLanguage # noqa: PLC0415
        checked_at = self.checked_at.isoformat()

        counts = self.counts.to_dict()

        duration_ms = self.duration_ms

        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        min_text_px = self.min_text_px

        mode = self.mode.value

        moderate = self.moderate

        policy = self.policy.value

        serious = self.serious


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkedAt": checked_at,
            "counts": counts,
            "durationMs": duration_ms,
            "issues": issues,
            "languages": languages,
            "minTextPx": min_text_px,
            "mode": mode,
            "moderate": moderate,
            "policy": policy,
            "serious": serious,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.accessibility_counts import AccessibilityCounts # noqa: PLC0415
        from ..models.accessibility_issue import AccessibilityIssue # noqa: PLC0415
        from ..models.accessibility_language import AccessibilityLanguage # noqa: PLC0415
        d = dict(src_dict)
        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        counts = AccessibilityCounts.from_dict(d.pop("counts"))




        duration_ms = d.pop("durationMs")

        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = AccessibilityIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = AccessibilityLanguage.from_dict(languages_item_data)



            languages.append(languages_item)


        min_text_px = d.pop("minTextPx")

        mode = AccessibilityReportMode(d.pop("mode"))




        moderate = d.pop("moderate")

        policy = AccessibilityReportPolicy(d.pop("policy"))




        serious = d.pop("serious")

        accessibility_report = cls(
            checked_at=checked_at,
            counts=counts,
            duration_ms=duration_ms,
            issues=issues,
            languages=languages,
            min_text_px=min_text_px,
            mode=mode,
            moderate=moderate,
            policy=policy,
            serious=serious,
        )

        return accessibility_report

