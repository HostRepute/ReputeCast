from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteIdentity")



@_attrs_define
class WebsiteIdentity:
    """ 
        Attributes:
            description (str):
            social_image (str):
            tagline (str):
            title (str):
     """

    description: str
    social_image: str
    tagline: str
    title: str





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        social_image = self.social_image

        tagline = self.tagline

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "socialImage": social_image,
            "tagline": tagline,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        social_image = d.pop("socialImage")

        tagline = d.pop("tagline")

        title = d.pop("title")

        website_identity = cls(
            description=description,
            social_image=social_image,
            tagline=tagline,
            title=title,
        )

        return website_identity

