from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ComposerPresent")



@_attrs_define
class ComposerPresent:
    """ 
        Attributes:
            color (str): Avatar colour
            editing (bool):
            name (str):
            seen_at (datetime.datetime):
            user_id (str):
     """

    color: str
    editing: bool
    name: str
    seen_at: datetime.datetime
    user_id: str





    def to_dict(self) -> dict[str, Any]:
        color = self.color

        editing = self.editing

        name = self.name

        seen_at = self.seen_at.isoformat()

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "editing": editing,
            "name": name,
            "seenAt": seen_at,
            "userId": user_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = d.pop("color")

        editing = d.pop("editing")

        name = d.pop("name")

        seen_at = datetime.datetime.fromisoformat(d.pop("seenAt"))




        user_id = d.pop("userId")

        composer_present = cls(
            color=color,
            editing=editing,
            name=name,
            seen_at=seen_at,
            user_id=user_id,
        )

        return composer_present

