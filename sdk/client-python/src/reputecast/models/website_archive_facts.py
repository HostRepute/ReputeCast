from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.website_archive_list import WebsiteArchiveList





T = TypeVar("T", bound="WebsiteArchiveFacts")



@_attrs_define
class WebsiteArchiveFacts:
    """ 
        Attributes:
            enabled (bool):
            feed (str):
            index (bool):
            issues (int):
            lists (list[WebsiteArchiveList]):
            since (int):
            tags (list[str]):
            url (str):
     """

    enabled: bool
    feed: str
    index: bool
    issues: int
    lists: list[WebsiteArchiveList]
    since: int
    tags: list[str]
    url: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_archive_list import WebsiteArchiveList # noqa: PLC0415
        enabled = self.enabled

        feed = self.feed

        index = self.index

        issues = self.issues

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        since = self.since

        tags = self.tags



        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "feed": feed,
            "index": index,
            "issues": issues,
            "lists": lists,
            "since": since,
            "tags": tags,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_archive_list import WebsiteArchiveList # noqa: PLC0415
        d = dict(src_dict)
        enabled = d.pop("enabled")

        feed = d.pop("feed")

        index = d.pop("index")

        issues = d.pop("issues")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = WebsiteArchiveList.from_dict(lists_item_data)



            lists.append(lists_item)


        since = d.pop("since")

        tags = cast(list[str], d.pop("tags"))


        url = d.pop("url")

        website_archive_facts = cls(
            enabled=enabled,
            feed=feed,
            index=index,
            issues=issues,
            lists=lists,
            since=since,
            tags=tags,
            url=url,
        )

        return website_archive_facts

