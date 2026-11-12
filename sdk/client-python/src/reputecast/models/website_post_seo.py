from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsitePostSEO")



@_attrs_define
class WebsitePostSEO:
    """ 
        Attributes:
            canonical_url (str):
            description (str):
            hide_from_search (bool):
            social_image (str):
            title (str):
     """

    canonical_url: str
    description: str
    hide_from_search: bool
    social_image: str
    title: str





    def to_dict(self) -> dict[str, Any]:
        canonical_url = self.canonical_url

        description = self.description

        hide_from_search = self.hide_from_search

        social_image = self.social_image

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canonicalUrl": canonical_url,
            "description": description,
            "hideFromSearch": hide_from_search,
            "socialImage": social_image,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        canonical_url = d.pop("canonicalUrl")

        description = d.pop("description")

        hide_from_search = d.pop("hideFromSearch")

        social_image = d.pop("socialImage")

        title = d.pop("title")

        website_post_seo = cls(
            canonical_url=canonical_url,
            description=description,
            hide_from_search=hide_from_search,
            social_image=social_image,
            title=title,
        )

        return website_post_seo

