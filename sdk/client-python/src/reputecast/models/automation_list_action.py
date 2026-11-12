from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_list_action_kind import AutomationListActionKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationListAction")



@_attrs_define
class AutomationListAction:
    """ 
        Attributes:
            kind (AutomationListActionKind):
            list_id (str | Unset):
            tag (str | Unset):
     """

    kind: AutomationListActionKind
    list_id: str | Unset = UNSET
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        list_id = self.list_id

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = AutomationListActionKind(d.pop("kind"))




        list_id = d.pop("listId", UNSET)

        tag = d.pop("tag", UNSET)

        automation_list_action = cls(
            kind=kind,
            list_id=list_id,
            tag=tag,
        )

        return automation_list_action

