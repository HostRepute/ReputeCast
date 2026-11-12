from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="GrowPreviewContact")



@_attrs_define
class GrowPreviewContact:
    """ 
        Attributes:
            counted (int):
            email (str):
            id (str):
            lists (list[str]):
            name (str):
            on_list (bool):
     """

    counted: int
    email: str
    id: str
    lists: list[str]
    name: str
    on_list: bool





    def to_dict(self) -> dict[str, Any]:
        counted = self.counted

        email = self.email

        id = self.id

        lists = self.lists



        name = self.name

        on_list = self.on_list


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counted": counted,
            "email": email,
            "id": id,
            "lists": lists,
            "name": name,
            "onList": on_list,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        counted = d.pop("counted")

        email = d.pop("email")

        id = d.pop("id")

        lists = cast(list[str], d.pop("lists"))


        name = d.pop("name")

        on_list = d.pop("onList")

        grow_preview_contact = cls(
            counted=counted,
            email=email,
            id=id,
            lists=lists,
            name=name,
            on_list=on_list,
        )

        return grow_preview_contact

