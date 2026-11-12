from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.role_impact_key import RoleImpactKey
  from ..models.role_impact_member import RoleImpactMember
  from ..models.role_impact_pending import RoleImpactPending
  from ..models.role_locked_permission import RoleLockedPermission





T = TypeVar("T", bound="RoleImpact")



@_attrs_define
class RoleImpact:
    """ 
        Attributes:
            added (list[str]):
            added_sensitive (list[str]):
            agents (int): MCP agents that lose permissions
            api_keys (int): API keys that lose permissions
            approvals (str):
            grace_days (int):
            keys (list[RoleImpactKey]):
            locked (list[RoleLockedPermission]):
            members (list[RoleImpactMember]):
            members_affected (int):
            pending_approvals (int):
            pending_requests (list[RoleImpactPending]):
            removed (list[str]):
            removed_sensitive (list[str]):
            sensitive (str):
            summary (str):
     """

    added: list[str]
    added_sensitive: list[str]
    agents: int
    api_keys: int
    approvals: str
    grace_days: int
    keys: list[RoleImpactKey]
    locked: list[RoleLockedPermission]
    members: list[RoleImpactMember]
    members_affected: int
    pending_approvals: int
    pending_requests: list[RoleImpactPending]
    removed: list[str]
    removed_sensitive: list[str]
    sensitive: str
    summary: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.role_impact_key import RoleImpactKey # noqa: PLC0415
        from ..models.role_impact_member import RoleImpactMember # noqa: PLC0415
        from ..models.role_impact_pending import RoleImpactPending # noqa: PLC0415
        from ..models.role_locked_permission import RoleLockedPermission # noqa: PLC0415
        added = self.added



        added_sensitive = self.added_sensitive



        agents = self.agents

        api_keys = self.api_keys

        approvals = self.approvals

        grace_days = self.grace_days

        keys = []
        for keys_item_data in self.keys:
            keys_item = keys_item_data.to_dict()
            keys.append(keys_item)



        locked = []
        for locked_item_data in self.locked:
            locked_item = locked_item_data.to_dict()
            locked.append(locked_item)



        members = []
        for members_item_data in self.members:
            members_item = members_item_data.to_dict()
            members.append(members_item)



        members_affected = self.members_affected

        pending_approvals = self.pending_approvals

        pending_requests = []
        for pending_requests_item_data in self.pending_requests:
            pending_requests_item = pending_requests_item_data.to_dict()
            pending_requests.append(pending_requests_item)



        removed = self.removed



        removed_sensitive = self.removed_sensitive



        sensitive = self.sensitive

        summary = self.summary


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added": added,
            "addedSensitive": added_sensitive,
            "agents": agents,
            "apiKeys": api_keys,
            "approvals": approvals,
            "graceDays": grace_days,
            "keys": keys,
            "locked": locked,
            "members": members,
            "membersAffected": members_affected,
            "pendingApprovals": pending_approvals,
            "pendingRequests": pending_requests,
            "removed": removed,
            "removedSensitive": removed_sensitive,
            "sensitive": sensitive,
            "summary": summary,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.role_impact_key import RoleImpactKey # noqa: PLC0415
        from ..models.role_impact_member import RoleImpactMember # noqa: PLC0415
        from ..models.role_impact_pending import RoleImpactPending # noqa: PLC0415
        from ..models.role_locked_permission import RoleLockedPermission # noqa: PLC0415
        d = dict(src_dict)
        added = cast(list[str], d.pop("added"))


        added_sensitive = cast(list[str], d.pop("addedSensitive"))


        agents = d.pop("agents")

        api_keys = d.pop("apiKeys")

        approvals = d.pop("approvals")

        grace_days = d.pop("graceDays")

        keys = []
        _keys = d.pop("keys")
        for keys_item_data in (_keys):
            keys_item = RoleImpactKey.from_dict(keys_item_data)



            keys.append(keys_item)


        locked = []
        _locked = d.pop("locked")
        for locked_item_data in (_locked):
            locked_item = RoleLockedPermission.from_dict(locked_item_data)



            locked.append(locked_item)


        members = []
        _members = d.pop("members")
        for members_item_data in (_members):
            members_item = RoleImpactMember.from_dict(members_item_data)



            members.append(members_item)


        members_affected = d.pop("membersAffected")

        pending_approvals = d.pop("pendingApprovals")

        pending_requests = []
        _pending_requests = d.pop("pendingRequests")
        for pending_requests_item_data in (_pending_requests):
            pending_requests_item = RoleImpactPending.from_dict(pending_requests_item_data)



            pending_requests.append(pending_requests_item)


        removed = cast(list[str], d.pop("removed"))


        removed_sensitive = cast(list[str], d.pop("removedSensitive"))


        sensitive = d.pop("sensitive")

        summary = d.pop("summary")

        role_impact = cls(
            added=added,
            added_sensitive=added_sensitive,
            agents=agents,
            api_keys=api_keys,
            approvals=approvals,
            grace_days=grace_days,
            keys=keys,
            locked=locked,
            members=members,
            members_affected=members_affected,
            pending_approvals=pending_approvals,
            pending_requests=pending_requests,
            removed=removed,
            removed_sensitive=removed_sensitive,
            sensitive=sensitive,
            summary=summary,
        )

        return role_impact

