from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.removal_item_kind import RemovalItemKind
from ..models.removal_item_state import RemovalItemState
from ..models.removal_item_status import RemovalItemStatus
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="RemovalItem")



@_attrs_define
class RemovalItem:
    """ 
        Attributes:
            id (str):
            kind (RemovalItemKind):
            name (str):
            changed (int | Unset):
            languages (list[str] | Unset):
            live_version (int | Unset):
            state (RemovalItemState | Unset):
            status (RemovalItemStatus | Unset):
     """

    id: str
    kind: RemovalItemKind
    name: str
    changed: int | Unset = UNSET
    languages: list[str] | Unset = UNSET
    live_version: int | Unset = UNSET
    state: RemovalItemState | Unset = UNSET
    status: RemovalItemStatus | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind.value

        name = self.name

        changed = self.changed

        languages: list[str] | Unset = UNSET
        if not isinstance(self.languages, Unset):
            languages = self.languages



        live_version = self.live_version

        state: str | Unset = UNSET
        if not isinstance(self.state, Unset):
            state = self.state.value


        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
            "name": name,
        })
        if changed is not UNSET:
            field_dict["changed"] = changed
        if languages is not UNSET:
            field_dict["languages"] = languages
        if live_version is not UNSET:
            field_dict["liveVersion"] = live_version
        if state is not UNSET:
            field_dict["state"] = state
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = RemovalItemKind(d.pop("kind"))




        name = d.pop("name")

        changed = d.pop("changed", UNSET)

        languages = cast(list[str], d.pop("languages", UNSET))


        live_version = d.pop("liveVersion", UNSET)

        _state = d.pop("state", UNSET)
        state: RemovalItemState | Unset
        if isinstance(_state,  Unset):
            state = UNSET
        else:
            state = RemovalItemState(_state)




        _status = d.pop("status", UNSET)
        status: RemovalItemStatus | Unset
        if isinstance(_status,  Unset):
            status = UNSET
        else:
            status = RemovalItemStatus(_status)




        removal_item = cls(
            id=id,
            kind=kind,
            name=name,
            changed=changed,
            languages=languages,
            live_version=live_version,
            state=state,
            status=status,
        )

        return removal_item

