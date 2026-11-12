from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.role_impact_key_kind import RoleImpactKeyKind
from typing import cast






T = TypeVar("T", bound="RoleImpactKey")



@_attrs_define
class RoleImpactKey:
    """ 
        Attributes:
            effect (str):
            id (str):
            kind (RoleImpactKeyKind):
            loses (list[str]):
            name (str):
     """

    effect: str
    id: str
    kind: RoleImpactKeyKind
    loses: list[str]
    name: str





    def to_dict(self) -> dict[str, Any]:
        effect = self.effect

        id = self.id

        kind = self.kind.value

        loses = self.loses



        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "effect": effect,
            "id": id,
            "kind": kind,
            "loses": loses,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        effect = d.pop("effect")

        id = d.pop("id")

        kind = RoleImpactKeyKind(d.pop("kind"))




        loses = cast(list[str], d.pop("loses"))


        name = d.pop("name")

        role_impact_key = cls(
            effect=effect,
            id=id,
            kind=kind,
            loses=loses,
            name=name,
        )

        return role_impact_key

