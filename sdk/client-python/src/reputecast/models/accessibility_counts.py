from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_counts_dir import AccessibilityCountsDir
from typing import cast






T = TypeVar("T", bound="AccessibilityCounts")



@_attrs_define
class AccessibilityCounts:
    """ 
        Attributes:
            data_tables (int):
            dir_ (AccessibilityCountsDir):
            images (int):
            images_ok (int):
            languages (list[str]):
            layout_tables (int):
            links (int):
            links_ok (int):
            over_image (int):
            role_added (int):
     """

    data_tables: int
    dir_: AccessibilityCountsDir
    images: int
    images_ok: int
    languages: list[str]
    layout_tables: int
    links: int
    links_ok: int
    over_image: int
    role_added: int





    def to_dict(self) -> dict[str, Any]:
        data_tables = self.data_tables

        dir_ = self.dir_.value

        images = self.images

        images_ok = self.images_ok

        languages = self.languages



        layout_tables = self.layout_tables

        links = self.links

        links_ok = self.links_ok

        over_image = self.over_image

        role_added = self.role_added


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "dataTables": data_tables,
            "dir": dir_,
            "images": images,
            "imagesOk": images_ok,
            "languages": languages,
            "layoutTables": layout_tables,
            "links": links,
            "linksOk": links_ok,
            "overImage": over_image,
            "roleAdded": role_added,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        data_tables = d.pop("dataTables")

        dir_ = AccessibilityCountsDir(d.pop("dir"))




        images = d.pop("images")

        images_ok = d.pop("imagesOk")

        languages = cast(list[str], d.pop("languages"))


        layout_tables = d.pop("layoutTables")

        links = d.pop("links")

        links_ok = d.pop("linksOk")

        over_image = d.pop("overImage")

        role_added = d.pop("roleAdded")

        accessibility_counts = cls(
            data_tables=data_tables,
            dir_=dir_,
            images=images,
            images_ok=images_ok,
            languages=languages,
            layout_tables=layout_tables,
            links=links,
            links_ok=links_ok,
            over_image=over_image,
            role_added=role_added,
        )

        return accessibility_counts

