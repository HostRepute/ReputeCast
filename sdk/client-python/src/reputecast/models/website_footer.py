from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteFooter")



@_attrs_define
class WebsiteFooter:
    """ 
        Attributes:
            archive (bool):
            preferences (bool):
            rss (bool):
     """

    archive: bool
    preferences: bool
    rss: bool





    def to_dict(self) -> dict[str, Any]:
        archive = self.archive

        preferences = self.preferences

        rss = self.rss


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "archive": archive,
            "preferences": preferences,
            "rss": rss,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        archive = d.pop("archive")

        preferences = d.pop("preferences")

        rss = d.pop("rss")

        website_footer = cls(
            archive=archive,
            preferences=preferences,
            rss=rss,
        )

        return website_footer

