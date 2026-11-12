from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PageTemplateCount")



@_attrs_define
class PageTemplateCount:
    """ 
        Attributes:
            count (int):
            template (str):
     """

    count: int
    template: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        template = self.template


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "template": template,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        template = d.pop("template")

        page_template_count = cls(
            count=count,
            template=template,
        )

        return page_template_count

