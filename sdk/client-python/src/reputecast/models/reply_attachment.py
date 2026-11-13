from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReplyAttachment")



@_attrs_define
class ReplyAttachment:
    """ 
        Attributes:
            name (str):
            size (int):
            type_ (str):
     """

    name: str
    size: int
    type_: str





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        size = self.size

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "size": size,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        size = d.pop("size")

        type_ = d.pop("type")

        reply_attachment = cls(
            name=name,
            size=size,
            type_=type_,
        )

        return reply_attachment

