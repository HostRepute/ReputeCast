from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TeamCounts")



@_attrs_define
class TeamCounts:
    """ 
        Attributes:
            active (int):
            deactivated (int):
            invited (int):
            members (int):
     """

    active: int
    deactivated: int
    invited: int
    members: int





    def to_dict(self) -> dict[str, Any]:
        active = self.active

        deactivated = self.deactivated

        invited = self.invited

        members = self.members


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "deactivated": deactivated,
            "invited": invited,
            "members": members,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active = d.pop("active")

        deactivated = d.pop("deactivated")

        invited = d.pop("invited")

        members = d.pop("members")

        team_counts = cls(
            active=active,
            deactivated=deactivated,
            invited=invited,
            members=members,
        )

        return team_counts

