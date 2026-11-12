from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BlockHiddenRec")



@_attrs_define
class BlockHiddenRec:
    """ 
        Attributes:
            id (str):
            initials (str):
            name (str):
            paused_by (str):
            state (str):
     """

    id: str
    initials: str
    name: str
    paused_by: str
    state: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        initials = self.initials

        name = self.name

        paused_by = self.paused_by

        state = self.state


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "initials": initials,
            "name": name,
            "pausedBy": paused_by,
            "state": state,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        initials = d.pop("initials")

        name = d.pop("name")

        paused_by = d.pop("pausedBy")

        state = d.pop("state")

        block_hidden_rec = cls(
            id=id,
            initials=initials,
            name=name,
            paused_by=paused_by,
            state=state,
        )

        return block_hidden_rec

