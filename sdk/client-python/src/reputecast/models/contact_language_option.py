from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ContactLanguageOption")



@_attrs_define
class ContactLanguageOption:
    """ 
        Attributes:
            code (str):
            name (str):
            offered (bool):
     """

    code: str
    name: str
    offered: bool





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        name = self.name

        offered = self.offered


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "name": name,
            "offered": offered,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        name = d.pop("name")

        offered = d.pop("offered")

        contact_language_option = cls(
            code=code,
            name=name,
            offered=offered,
        )

        return contact_language_option

