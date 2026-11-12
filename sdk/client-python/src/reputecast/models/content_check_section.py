from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ContentCheckSection")



@_attrs_define
class ContentCheckSection:
    """ 
        Attributes:
            images (int):
            images_with_alt (int):
            index (int):
            issues (int):
            label (str):
            links (int):
            links_ok (int):
     """

    images: int
    images_with_alt: int
    index: int
    issues: int
    label: str
    links: int
    links_ok: int





    def to_dict(self) -> dict[str, Any]:
        images = self.images

        images_with_alt = self.images_with_alt

        index = self.index

        issues = self.issues

        label = self.label

        links = self.links

        links_ok = self.links_ok


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "images": images,
            "imagesWithAlt": images_with_alt,
            "index": index,
            "issues": issues,
            "label": label,
            "links": links,
            "linksOk": links_ok,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        images = d.pop("images")

        images_with_alt = d.pop("imagesWithAlt")

        index = d.pop("index")

        issues = d.pop("issues")

        label = d.pop("label")

        links = d.pop("links")

        links_ok = d.pop("linksOk")

        content_check_section = cls(
            images=images,
            images_with_alt=images_with_alt,
            index=index,
            issues=issues,
            label=label,
            links=links,
            links_ok=links_ok,
        )

        return content_check_section

