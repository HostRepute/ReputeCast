from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.archive_issue_item import ArchiveIssueItem
  from ..models.archive_list_count import ArchiveListCount





T = TypeVar("T", bound="ArchiveIssuePage")



@_attrs_define
class ArchiveIssuePage:
    """ 
        Attributes:
            issues (list[ArchiveIssueItem]):
            lists (list[ArchiveListCount]):
            page (int):
            page_size (int):
            published (int):
            total (int):
     """

    issues: list[ArchiveIssueItem]
    lists: list[ArchiveListCount]
    page: int
    page_size: int
    published: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.archive_issue_item import ArchiveIssueItem # noqa: PLC0415
        from ..models.archive_list_count import ArchiveListCount # noqa: PLC0415
        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        page = self.page

        page_size = self.page_size

        published = self.published

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "issues": issues,
            "lists": lists,
            "page": page,
            "pageSize": page_size,
            "published": published,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.archive_issue_item import ArchiveIssueItem # noqa: PLC0415
        from ..models.archive_list_count import ArchiveListCount # noqa: PLC0415
        d = dict(src_dict)
        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = ArchiveIssueItem.from_dict(issues_item_data)



            issues.append(issues_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ArchiveListCount.from_dict(lists_item_data)



            lists.append(lists_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        published = d.pop("published")

        total = d.pop("total")

        archive_issue_page = cls(
            issues=issues,
            lists=lists,
            page=page,
            page_size=page_size,
            published=published,
            total=total,
        )

        return archive_issue_page

