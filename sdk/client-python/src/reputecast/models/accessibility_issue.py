from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_issue_kind import AccessibilityIssueKind
from ..models.accessibility_issue_severity import AccessibilityIssueSeverity
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.accessibility_element import AccessibilityElement
  from ..models.accessibility_fix_spec import AccessibilityFixSpec
  from ..models.accessibility_issue_version import AccessibilityIssueVersion





T = TypeVar("T", bound="AccessibilityIssue")



@_attrs_define
class AccessibilityIssue:
    """ 
        Attributes:
            element (AccessibilityElement):
            fix (AccessibilityFixSpec):
            id (str):
            kind (AccessibilityIssueKind):
            languages (list[str]):
            location (list[str]):
            path (list[int]):
            paths (list[list[int]]):
            severity (AccessibilityIssueSeverity):
            short (str):
            tag (str):
            title (str):
            versions (list[AccessibilityIssueVersion]):
            detail (str | Unset):
            name (str | Unset):
     """

    element: AccessibilityElement
    fix: AccessibilityFixSpec
    id: str
    kind: AccessibilityIssueKind
    languages: list[str]
    location: list[str]
    path: list[int]
    paths: list[list[int]]
    severity: AccessibilityIssueSeverity
    short: str
    tag: str
    title: str
    versions: list[AccessibilityIssueVersion]
    detail: str | Unset = UNSET
    name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.accessibility_element import AccessibilityElement # noqa: PLC0415
        from ..models.accessibility_fix_spec import AccessibilityFixSpec # noqa: PLC0415
        from ..models.accessibility_issue_version import AccessibilityIssueVersion # noqa: PLC0415
        element = self.element.to_dict()

        fix = self.fix.to_dict()

        id = self.id

        kind = self.kind.value

        languages = self.languages



        location = self.location



        path = self.path



        paths = []
        for paths_item_data in self.paths:
            paths_item = paths_item_data


            paths.append(paths_item)



        severity = self.severity.value

        short = self.short

        tag = self.tag

        title = self.title

        versions = []
        for versions_item_data in self.versions:
            versions_item = versions_item_data.to_dict()
            versions.append(versions_item)



        detail = self.detail

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "element": element,
            "fix": fix,
            "id": id,
            "kind": kind,
            "languages": languages,
            "location": location,
            "path": path,
            "paths": paths,
            "severity": severity,
            "short": short,
            "tag": tag,
            "title": title,
            "versions": versions,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.accessibility_element import AccessibilityElement # noqa: PLC0415
        from ..models.accessibility_fix_spec import AccessibilityFixSpec # noqa: PLC0415
        from ..models.accessibility_issue_version import AccessibilityIssueVersion # noqa: PLC0415
        d = dict(src_dict)
        element = AccessibilityElement.from_dict(d.pop("element"))




        fix = AccessibilityFixSpec.from_dict(d.pop("fix"))




        id = d.pop("id")

        kind = AccessibilityIssueKind(d.pop("kind"))




        languages = cast(list[str], d.pop("languages"))


        location = cast(list[str], d.pop("location"))


        path = cast(list[int], d.pop("path"))


        paths = []
        _paths = d.pop("paths")
        for paths_item_data in (_paths):
            paths_item = cast(list[int], paths_item_data)

            paths.append(paths_item)


        severity = AccessibilityIssueSeverity(d.pop("severity"))




        short = d.pop("short")

        tag = d.pop("tag")

        title = d.pop("title")

        versions = []
        _versions = d.pop("versions")
        for versions_item_data in (_versions):
            versions_item = AccessibilityIssueVersion.from_dict(versions_item_data)



            versions.append(versions_item)


        detail = d.pop("detail", UNSET)

        name = d.pop("name", UNSET)

        accessibility_issue = cls(
            element=element,
            fix=fix,
            id=id,
            kind=kind,
            languages=languages,
            location=location,
            path=path,
            paths=paths,
            severity=severity,
            short=short,
            tag=tag,
            title=title,
            versions=versions,
            detail=detail,
            name=name,
        )

        return accessibility_issue

