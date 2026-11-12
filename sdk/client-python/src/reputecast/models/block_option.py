from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.block_option_kind import BlockOptionKind






T = TypeVar("T", bound="BlockOption")



@_attrs_define
class BlockOption:
    """ 
        Attributes:
            hidden (bool):
            id (str):
            initials (str):
            kind (BlockOptionKind):
            name (str):
            show_block (bool):
            state (str):
     """

    hidden: bool
    id: str
    initials: str
    kind: BlockOptionKind
    name: str
    show_block: bool
    state: str





    def to_dict(self) -> dict[str, Any]:
        hidden = self.hidden

        id = self.id

        initials = self.initials

        kind = self.kind.value

        name = self.name

        show_block = self.show_block

        state = self.state


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hidden": hidden,
            "id": id,
            "initials": initials,
            "kind": kind,
            "name": name,
            "showBlock": show_block,
            "state": state,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hidden = d.pop("hidden")

        id = d.pop("id")

        initials = d.pop("initials")

        kind = BlockOptionKind(d.pop("kind"))




        name = d.pop("name")

        show_block = d.pop("showBlock")

        state = d.pop("state")

        block_option = cls(
            hidden=hidden,
            id=id,
            initials=initials,
            kind=kind,
            name=name,
            show_block=show_block,
            state=state,
        )

        return block_option

