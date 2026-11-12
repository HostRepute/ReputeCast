from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PageCounts")



@_attrs_define
class PageCounts:
    """ 
        Attributes:
            blocks (int):
            images (int):
            images_with_alt (int):
            links (int):
     """

    blocks: int
    images: int
    images_with_alt: int
    links: int





    def to_dict(self) -> dict[str, Any]:
        blocks = self.blocks

        images = self.images

        images_with_alt = self.images_with_alt

        links = self.links


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "images": images,
            "imagesWithAlt": images_with_alt,
            "links": links,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocks = d.pop("blocks")

        images = d.pop("images")

        images_with_alt = d.pop("imagesWithAlt")

        links = d.pop("links")

        page_counts = cls(
            blocks=blocks,
            images=images,
            images_with_alt=images_with_alt,
            links=links,
        )

        return page_counts

