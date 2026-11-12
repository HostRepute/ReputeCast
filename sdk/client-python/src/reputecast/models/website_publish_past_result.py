from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsitePublishPastResult")



@_attrs_define
class WebsitePublishPastResult:
    """ 
        Attributes:
            published (int):
     """

    published: int





    def to_dict(self) -> dict[str, Any]:
        published = self.published


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "published": published,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        published = d.pop("published")

        website_publish_past_result = cls(
            published=published,
        )

        return website_publish_past_result

