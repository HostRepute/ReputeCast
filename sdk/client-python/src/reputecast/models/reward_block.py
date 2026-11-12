from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RewardBlock")



@_attrs_define
class RewardBlock:
    """ 
        Attributes:
            button (str):
            description (bool):
            expiry (bool):
            hub (bool):
            image (bool):
     """

    button: str
    description: bool
    expiry: bool
    hub: bool
    image: bool





    def to_dict(self) -> dict[str, Any]:
        button = self.button

        description = self.description

        expiry = self.expiry

        hub = self.hub

        image = self.image


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "button": button,
            "description": description,
            "expiry": expiry,
            "hub": hub,
            "image": image,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        button = d.pop("button")

        description = d.pop("description")

        expiry = d.pop("expiry")

        hub = d.pop("hub")

        image = d.pop("image")

        reward_block = cls(
            button=button,
            description=description,
            expiry=expiry,
            hub=hub,
            image=image,
        )

        return reward_block

