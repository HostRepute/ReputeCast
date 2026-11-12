from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RecListOption")



@_attrs_define
class RecListOption:
    """ 
        Attributes:
            id (str):
            name (str):
            subscribers (int):
     """

    id: str
    name: str
    subscribers: int





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        subscribers = self.subscribers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "subscribers": subscribers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        subscribers = d.pop("subscribers")

        rec_list_option = cls(
            id=id,
            name=name,
            subscribers=subscribers,
        )

        return rec_list_option

