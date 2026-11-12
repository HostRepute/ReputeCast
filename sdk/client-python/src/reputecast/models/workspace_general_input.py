from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkspaceGeneralInput")



@_attrs_define
class WorkspaceGeneralInput:
    """ 
        Attributes:
            name (str):
            timezone (str):
            default_sender_id (str | Unset):
     """

    name: str
    timezone: str
    default_sender_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        timezone = self.timezone

        default_sender_id = self.default_sender_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "timezone": timezone,
        })
        if default_sender_id is not UNSET:
            field_dict["defaultSenderId"] = default_sender_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        timezone = d.pop("timezone")

        default_sender_id = d.pop("defaultSenderId", UNSET)

        workspace_general_input = cls(
            name=name,
            timezone=timezone,
            default_sender_id=default_sender_id,
        )

        return workspace_general_input

