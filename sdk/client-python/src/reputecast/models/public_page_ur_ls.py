from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PublicPageURLs")



@_attrs_define
class PublicPageURLs:
    """ 
        Attributes:
            archive (str):
            feed (str):
            preferences (str): Asks for a preference link by email
            sitemap (str):
            subscribe (str):
     """

    archive: str
    feed: str
    preferences: str
    sitemap: str
    subscribe: str





    def to_dict(self) -> dict[str, Any]:
        archive = self.archive

        feed = self.feed

        preferences = self.preferences

        sitemap = self.sitemap

        subscribe = self.subscribe


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "archive": archive,
            "feed": feed,
            "preferences": preferences,
            "sitemap": sitemap,
            "subscribe": subscribe,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        archive = d.pop("archive")

        feed = d.pop("feed")

        preferences = d.pop("preferences")

        sitemap = d.pop("sitemap")

        subscribe = d.pop("subscribe")

        public_page_ur_ls = cls(
            archive=archive,
            feed=feed,
            preferences=preferences,
            sitemap=sitemap,
            subscribe=subscribe,
        )

        return public_page_ur_ls

