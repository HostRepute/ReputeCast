from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ArchiveIssueInput")



@_attrs_define
class ArchiveIssueInput:
    """ 
        Attributes:
            excerpt (str): Empty uses the preheader
            hidden (bool): Hide the issue from the archive, the feed and the sitemap
            label (str):  Example: #141.
            list_id (str): The list the issue is filed under
            title (str): Empty uses the subject
     """

    excerpt: str
    hidden: bool
    label: str
    list_id: str
    title: str





    def to_dict(self) -> dict[str, Any]:
        excerpt = self.excerpt

        hidden = self.hidden

        label = self.label

        list_id = self.list_id

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "excerpt": excerpt,
            "hidden": hidden,
            "label": label,
            "listId": list_id,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        excerpt = d.pop("excerpt")

        hidden = d.pop("hidden")

        label = d.pop("label")

        list_id = d.pop("listId")

        title = d.pop("title")

        archive_issue_input = cls(
            excerpt=excerpt,
            hidden=hidden,
            label=label,
            list_id=list_id,
            title=title,
        )

        return archive_issue_input

