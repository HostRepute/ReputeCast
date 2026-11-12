from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_graph_issue import AutomationGraphIssue





T = TypeVar("T", bound="AutomationDraftSaved")



@_attrs_define
class AutomationDraftSaved:
    """ 
        Attributes:
            changes (int): Unpublished changes compared with the live version
            draft_version (int):
            issues (list[AutomationGraphIssue]):
            revision (int):
     """

    changes: int
    draft_version: int
    issues: list[AutomationGraphIssue]
    revision: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_graph_issue import AutomationGraphIssue # noqa: PLC0415
        changes = self.changes

        draft_version = self.draft_version

        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        revision = self.revision


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "draftVersion": draft_version,
            "issues": issues,
            "revision": revision,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_graph_issue import AutomationGraphIssue # noqa: PLC0415
        d = dict(src_dict)
        changes = d.pop("changes")

        draft_version = d.pop("draftVersion")

        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = AutomationGraphIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        revision = d.pop("revision")

        automation_draft_saved = cls(
            changes=changes,
            draft_version=draft_version,
            issues=issues,
            revision=revision,
        )

        return automation_draft_saved

