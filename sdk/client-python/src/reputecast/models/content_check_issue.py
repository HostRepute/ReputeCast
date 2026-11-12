from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.content_check_issue_group import ContentCheckIssueGroup
from ..models.content_check_issue_severity import ContentCheckIssueSeverity
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.content_check_fix import ContentCheckFix





T = TypeVar("T", bound="ContentCheckIssue")



@_attrs_define
class ContentCheckIssue:
    """ 
        Attributes:
            fixes (list[ContentCheckFix]):
            group (ContentCheckIssueGroup):
            id (str):  Example: links.blocklist.
            path (list[int]):
            section (int):
            severity (ContentCheckIssueSeverity):
            title (str):
            detail (str | Unset):
            tag (str | Unset):
            url (str | Unset):
     """

    fixes: list[ContentCheckFix]
    group: ContentCheckIssueGroup
    id: str
    path: list[int]
    section: int
    severity: ContentCheckIssueSeverity
    title: str
    detail: str | Unset = UNSET
    tag: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.content_check_fix import ContentCheckFix # noqa: PLC0415
        fixes = []
        for fixes_item_data in self.fixes:
            fixes_item = fixes_item_data.to_dict()
            fixes.append(fixes_item)



        group = self.group.value

        id = self.id

        path = self.path



        section = self.section

        severity = self.severity.value

        title = self.title

        detail = self.detail

        tag = self.tag

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fixes": fixes,
            "group": group,
            "id": id,
            "path": path,
            "section": section,
            "severity": severity,
            "title": title,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if tag is not UNSET:
            field_dict["tag"] = tag
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.content_check_fix import ContentCheckFix # noqa: PLC0415
        d = dict(src_dict)
        fixes = []
        _fixes = d.pop("fixes")
        for fixes_item_data in (_fixes):
            fixes_item = ContentCheckFix.from_dict(fixes_item_data)



            fixes.append(fixes_item)


        group = ContentCheckIssueGroup(d.pop("group"))




        id = d.pop("id")

        path = cast(list[int], d.pop("path"))


        section = d.pop("section")

        severity = ContentCheckIssueSeverity(d.pop("severity"))




        title = d.pop("title")

        detail = d.pop("detail", UNSET)

        tag = d.pop("tag", UNSET)

        url = d.pop("url", UNSET)

        content_check_issue = cls(
            fixes=fixes,
            group=group,
            id=id,
            path=path,
            section=section,
            severity=severity,
            title=title,
            detail=detail,
            tag=tag,
            url=url,
        )

        return content_check_issue

