from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PageContentInput")



@_attrs_define
class PageContentInput:
    """ 
        Attributes:
            base_version (int):
            html (str):
            force (bool | Unset):
            manual (bool | Unset):
            project (Any | Unset):
     """

    base_version: int
    html: str
    force: bool | Unset = UNSET
    manual: bool | Unset = UNSET
    project: Any | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        base_version = self.base_version

        html = self.html

        force = self.force

        manual = self.manual

        project = self.project


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseVersion": base_version,
            "html": html,
        })
        if force is not UNSET:
            field_dict["force"] = force
        if manual is not UNSET:
            field_dict["manual"] = manual
        if project is not UNSET:
            field_dict["project"] = project

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base_version = d.pop("baseVersion")

        html = d.pop("html")

        force = d.pop("force", UNSET)

        manual = d.pop("manual", UNSET)

        project = d.pop("project", UNSET)

        page_content_input = cls(
            base_version=base_version,
            html=html,
            force=force,
            manual=manual,
            project=project,
        )

        return page_content_input

