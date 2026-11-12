from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PushPromptCreate")



@_attrs_define
class PushPromptCreate:
    """ 
        Attributes:
            name (str):
            site_id (str | Unset):
     """

    name: str
    site_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        site_id = self.site_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if site_id is not UNSET:
            field_dict["siteId"] = site_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        site_id = d.pop("siteId", UNSET)

        push_prompt_create = cls(
            name=name,
            site_id=site_id,
        )

        return push_prompt_create

