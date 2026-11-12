from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteFeedURLs")



@_attrs_define
class WebsiteFeedURLs:
    """ 
        Attributes:
            archive_feed (str):
            feed (str):
            llms (str):
            sitemap (str):
            tag_feed (str):
     """

    archive_feed: str
    feed: str
    llms: str
    sitemap: str
    tag_feed: str





    def to_dict(self) -> dict[str, Any]:
        archive_feed = self.archive_feed

        feed = self.feed

        llms = self.llms

        sitemap = self.sitemap

        tag_feed = self.tag_feed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "archiveFeed": archive_feed,
            "feed": feed,
            "llms": llms,
            "sitemap": sitemap,
            "tagFeed": tag_feed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        archive_feed = d.pop("archiveFeed")

        feed = d.pop("feed")

        llms = d.pop("llms")

        sitemap = d.pop("sitemap")

        tag_feed = d.pop("tagFeed")

        website_feed_ur_ls = cls(
            archive_feed=archive_feed,
            feed=feed,
            llms=llms,
            sitemap=sitemap,
            tag_feed=tag_feed,
        )

        return website_feed_ur_ls

