from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audit_actor_type import AuditActorType
from ..types import UNSET, Unset






T = TypeVar("T", bound="AuditActor")



@_attrs_define
class AuditActor:
    """ 
        Attributes:
            label (str):
            name (str):
            type_ (AuditActorType):
            avatar_color (str | Unset):
            id (str | Unset):
            role (str | Unset):
     """

    label: str
    name: str
    type_: AuditActorType
    avatar_color: str | Unset = UNSET
    id: str | Unset = UNSET
    role: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        name = self.name

        type_ = self.type_.value

        avatar_color = self.avatar_color

        id = self.id

        role = self.role


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "name": name,
            "type": type_,
        })
        if avatar_color is not UNSET:
            field_dict["avatarColor"] = avatar_color
        if id is not UNSET:
            field_dict["id"] = id
        if role is not UNSET:
            field_dict["role"] = role

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        name = d.pop("name")

        type_ = AuditActorType(d.pop("type"))




        avatar_color = d.pop("avatarColor", UNSET)

        id = d.pop("id", UNSET)

        role = d.pop("role", UNSET)

        audit_actor = cls(
            label=label,
            name=name,
            type_=type_,
            avatar_color=avatar_color,
            id=id,
            role=role,
        )

        return audit_actor

