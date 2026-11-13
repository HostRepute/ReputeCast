from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.reply_source_type import ReplySourceType






T = TypeVar("T", bound="ReplySource")



@_attrs_define
class ReplySource:
    """ 
        Attributes:
            id (str):
            name (str):
            type_ (ReplySourceType):
     """

    id: str
    name: str
    type_: ReplySourceType





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        type_ = self.type_.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        type_ = ReplySourceType(d.pop("type"))




        reply_source = cls(
            id=id,
            name=name,
            type_=type_,
        )

        return reply_source

