from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_create_template import PageCreateTemplate
from ..types import UNSET, Unset






T = TypeVar("T", bound="PageCreate")



@_attrs_define
class PageCreate:
    """ 
        Attributes:
            name (str):
            template (PageCreateTemplate):
            slug (str | Unset):
     """

    name: str
    template: PageCreateTemplate
    slug: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        template = self.template.value

        slug = self.slug


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "template": template,
        })
        if slug is not UNSET:
            field_dict["slug"] = slug

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        template = PageCreateTemplate(d.pop("template"))




        slug = d.pop("slug", UNSET)

        page_create = cls(
            name=name,
            template=template,
            slug=slug,
        )

        return page_create

