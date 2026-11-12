from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_element_role import AccessibilityElementRole
from ..types import UNSET, Unset






T = TypeVar("T", bound="AccessibilityElement")



@_attrs_define
class AccessibilityElement:
    """ 
        Attributes:
            count (int):
            index (int):
            role (AccessibilityElementRole):
            tag (str):
            href (str | Unset):
            label (str | Unset):
            section (str | Unset):
            src (str | Unset):
     """

    count: int
    index: int
    role: AccessibilityElementRole
    tag: str
    href: str | Unset = UNSET
    label: str | Unset = UNSET
    section: str | Unset = UNSET
    src: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        index = self.index

        role = self.role.value

        tag = self.tag

        href = self.href

        label = self.label

        section = self.section

        src = self.src


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "index": index,
            "role": role,
            "tag": tag,
        })
        if href is not UNSET:
            field_dict["href"] = href
        if label is not UNSET:
            field_dict["label"] = label
        if section is not UNSET:
            field_dict["section"] = section
        if src is not UNSET:
            field_dict["src"] = src

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        index = d.pop("index")

        role = AccessibilityElementRole(d.pop("role"))




        tag = d.pop("tag")

        href = d.pop("href", UNSET)

        label = d.pop("label", UNSET)

        section = d.pop("section", UNSET)

        src = d.pop("src", UNSET)

        accessibility_element = cls(
            count=count,
            index=index,
            role=role,
            tag=tag,
            href=href,
            label=label,
            section=section,
            src=src,
        )

        return accessibility_element

