from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralListOption")



@_attrs_define
class ReferralListOption:
    """ 
        Attributes:
            description (str):
            id (str):
            name (str):
            subscribers (int):
            taken (bool):
     """

    description: str
    id: str
    name: str
    subscribers: int
    taken: bool





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        id = self.id

        name = self.name

        subscribers = self.subscribers

        taken = self.taken


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "id": id,
            "name": name,
            "subscribers": subscribers,
            "taken": taken,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        id = d.pop("id")

        name = d.pop("name")

        subscribers = d.pop("subscribers")

        taken = d.pop("taken")

        referral_list_option = cls(
            description=description,
            id=id,
            name=name,
            subscribers=subscribers,
            taken=taken,
        )

        return referral_list_option

