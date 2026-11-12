from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.team_counts import TeamCounts
  from ..models.team_member import TeamMember
  from ..models.team_role_ref import TeamRoleRef





T = TypeVar("T", bound="TeamMemberList")



@_attrs_define
class TeamMemberList:
    """ 
        Attributes:
            can_invite (bool):
            can_manage (bool):
            counts (TeamCounts):
            helpers (list[str]):
            items (list[TeamMember]):
            roles (list[TeamRoleRef]):
            reason (str | Unset):
     """

    can_invite: bool
    can_manage: bool
    counts: TeamCounts
    helpers: list[str]
    items: list[TeamMember]
    roles: list[TeamRoleRef]
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.team_counts import TeamCounts # noqa: PLC0415
        from ..models.team_member import TeamMember # noqa: PLC0415
        from ..models.team_role_ref import TeamRoleRef # noqa: PLC0415
        can_invite = self.can_invite

        can_manage = self.can_manage

        counts = self.counts.to_dict()

        helpers = self.helpers



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)



        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canInvite": can_invite,
            "canManage": can_manage,
            "counts": counts,
            "helpers": helpers,
            "items": items,
            "roles": roles,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.team_counts import TeamCounts # noqa: PLC0415
        from ..models.team_member import TeamMember # noqa: PLC0415
        from ..models.team_role_ref import TeamRoleRef # noqa: PLC0415
        d = dict(src_dict)
        can_invite = d.pop("canInvite")

        can_manage = d.pop("canManage")

        counts = TeamCounts.from_dict(d.pop("counts"))




        helpers = cast(list[str], d.pop("helpers"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = TeamMember.from_dict(items_item_data)



            items.append(items_item)


        roles = []
        _roles = d.pop("roles")
        for roles_item_data in (_roles):
            roles_item = TeamRoleRef.from_dict(roles_item_data)



            roles.append(roles_item)


        reason = d.pop("reason", UNSET)

        team_member_list = cls(
            can_invite=can_invite,
            can_manage=can_manage,
            counts=counts,
            helpers=helpers,
            items=items,
            roles=roles,
            reason=reason,
        )

        return team_member_list

