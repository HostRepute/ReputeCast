from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsitePostCreate")



@_attrs_define
class WebsitePostCreate:
    """ 
        Attributes:
            mjml (str | Unset):
            title (str | Unset):
     """

    mjml: str | Unset = UNSET
    title: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mjml = self.mjml

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mjml = d.pop("mjml", UNSET)

        title = d.pop("title", UNSET)

        website_post_create = cls(
            mjml=mjml,
            title=title,
        )

        return website_post_create

