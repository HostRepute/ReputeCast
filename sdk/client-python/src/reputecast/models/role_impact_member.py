from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.role_impact_member_two_factor import RoleImpactMemberTwoFactor
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="RoleImpactMember")



@_attrs_define
class RoleImpactMember:
    """ 
        Attributes:
            avatar_color (str):
            effect (str):
            email (str):
            gains (list[str]):
            id (str):
            loses (list[str]):
            name (str):
            two_factor (RoleImpactMemberTwoFactor | Unset):
     """

    avatar_color: str
    effect: str
    email: str
    gains: list[str]
    id: str
    loses: list[str]
    name: str
    two_factor: RoleImpactMemberTwoFactor | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        avatar_color = self.avatar_color

        effect = self.effect

        email = self.email

        gains = self.gains



        id = self.id

        loses = self.loses



        name = self.name

        two_factor: str | Unset = UNSET
        if not isinstance(self.two_factor, Unset):
            two_factor = self.two_factor.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "avatarColor": avatar_color,
            "effect": effect,
            "email": email,
            "gains": gains,
            "id": id,
            "loses": loses,
            "name": name,
        })
        if two_factor is not UNSET:
            field_dict["twoFactor"] = two_factor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        avatar_color = d.pop("avatarColor")

        effect = d.pop("effect")

        email = d.pop("email")

        gains = cast(list[str], d.pop("gains"))


        id = d.pop("id")

        loses = cast(list[str], d.pop("loses"))


        name = d.pop("name")

        _two_factor = d.pop("twoFactor", UNSET)
        two_factor: RoleImpactMemberTwoFactor | Unset
        if isinstance(_two_factor,  Unset):
            two_factor = UNSET
        else:
            two_factor = RoleImpactMemberTwoFactor(_two_factor)




        role_impact_member = cls(
            avatar_color=avatar_color,
            effect=effect,
            email=email,
            gains=gains,
            id=id,
            loses=loses,
            name=name,
            two_factor=two_factor,
        )

        return role_impact_member

