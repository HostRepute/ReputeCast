from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ContactTagInput")



@_attrs_define
class ContactTagInput:
    """ 
        Attributes:
            email (str):
            tags (list[str]):
     """

    email: str
    tags: list[str]





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        tags = self.tags




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "tags": tags,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        tags = cast(list[str], d.pop("tags"))


        contact_tag_input = cls(
            email=email,
            tags=tags,
        )

        return contact_tag_input

