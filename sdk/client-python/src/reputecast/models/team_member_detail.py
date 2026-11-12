from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.member_sign_in_method import MemberSignInMethod
  from ..models.team_effective_permission import TeamEffectivePermission
  from ..models.team_member import TeamMember
  from ..models.team_member_activity import TeamMemberActivity
  from ..models.team_member_api_key import TeamMemberAPIKey
  from ..models.team_member_approval import TeamMemberApproval





T = TypeVar("T", bound="TeamMemberDetail")



@_attrs_define
class TeamMemberDetail:
    """ 
        Attributes:
            activity (list[TeamMemberActivity]):
            api_keys (list[TeamMemberAPIKey]):
            approvals_given (list[TeamMemberApproval]):
            approvals_requested (list[TeamMemberApproval]):
            can_approve (bool):
            can_manage (bool):
            groups (list[str]):
            helpers (list[str]):
            member (TeamMember):
            permissions (list[TeamEffectivePermission]):
            scheduled_campaigns (int):
            sessions (int):
            sessions_shared (bool):
            sign_in_methods (list[MemberSignInMethod]):
            keys_note (str | Unset):
            sign_out_reason (str | Unset):
            sso_domain (str | Unset):
     """

    activity: list[TeamMemberActivity]
    api_keys: list[TeamMemberAPIKey]
    approvals_given: list[TeamMemberApproval]
    approvals_requested: list[TeamMemberApproval]
    can_approve: bool
    can_manage: bool
    groups: list[str]
    helpers: list[str]
    member: TeamMember
    permissions: list[TeamEffectivePermission]
    scheduled_campaigns: int
    sessions: int
    sessions_shared: bool
    sign_in_methods: list[MemberSignInMethod]
    keys_note: str | Unset = UNSET
    sign_out_reason: str | Unset = UNSET
    sso_domain: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.member_sign_in_method import MemberSignInMethod # noqa: PLC0415
        from ..models.team_effective_permission import TeamEffectivePermission # noqa: PLC0415
        from ..models.team_member import TeamMember # noqa: PLC0415
        from ..models.team_member_activity import TeamMemberActivity # noqa: PLC0415
        from ..models.team_member_api_key import TeamMemberAPIKey # noqa: PLC0415
        from ..models.team_member_approval import TeamMemberApproval # noqa: PLC0415
        activity = []
        for activity_item_data in self.activity:
            activity_item = activity_item_data.to_dict()
            activity.append(activity_item)



        api_keys = []
        for api_keys_item_data in self.api_keys:
            api_keys_item = api_keys_item_data.to_dict()
            api_keys.append(api_keys_item)



        approvals_given = []
        for approvals_given_item_data in self.approvals_given:
            approvals_given_item = approvals_given_item_data.to_dict()
            approvals_given.append(approvals_given_item)



        approvals_requested = []
        for approvals_requested_item_data in self.approvals_requested:
            approvals_requested_item = approvals_requested_item_data.to_dict()
            approvals_requested.append(approvals_requested_item)



        can_approve = self.can_approve

        can_manage = self.can_manage

        groups = self.groups



        helpers = self.helpers



        member = self.member.to_dict()

        permissions = []
        for permissions_item_data in self.permissions:
            permissions_item = permissions_item_data.to_dict()
            permissions.append(permissions_item)



        scheduled_campaigns = self.scheduled_campaigns

        sessions = self.sessions

        sessions_shared = self.sessions_shared

        sign_in_methods = []
        for sign_in_methods_item_data in self.sign_in_methods:
            sign_in_methods_item = sign_in_methods_item_data.to_dict()
            sign_in_methods.append(sign_in_methods_item)



        keys_note = self.keys_note

        sign_out_reason = self.sign_out_reason

        sso_domain = self.sso_domain


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "activity": activity,
            "apiKeys": api_keys,
            "approvalsGiven": approvals_given,
            "approvalsRequested": approvals_requested,
            "canApprove": can_approve,
            "canManage": can_manage,
            "groups": groups,
            "helpers": helpers,
            "member": member,
            "permissions": permissions,
            "scheduledCampaigns": scheduled_campaigns,
            "sessions": sessions,
            "sessionsShared": sessions_shared,
            "signInMethods": sign_in_methods,
        })
        if keys_note is not UNSET:
            field_dict["keysNote"] = keys_note
        if sign_out_reason is not UNSET:
            field_dict["signOutReason"] = sign_out_reason
        if sso_domain is not UNSET:
            field_dict["ssoDomain"] = sso_domain

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.member_sign_in_method import MemberSignInMethod # noqa: PLC0415
        from ..models.team_effective_permission import TeamEffectivePermission # noqa: PLC0415
        from ..models.team_member import TeamMember # noqa: PLC0415
        from ..models.team_member_activity import TeamMemberActivity # noqa: PLC0415
        from ..models.team_member_api_key import TeamMemberAPIKey # noqa: PLC0415
        from ..models.team_member_approval import TeamMemberApproval # noqa: PLC0415
        d = dict(src_dict)
        activity = []
        _activity = d.pop("activity")
        for activity_item_data in (_activity):
            activity_item = TeamMemberActivity.from_dict(activity_item_data)



            activity.append(activity_item)


        api_keys = []
        _api_keys = d.pop("apiKeys")
        for api_keys_item_data in (_api_keys):
            api_keys_item = TeamMemberAPIKey.from_dict(api_keys_item_data)



            api_keys.append(api_keys_item)


        approvals_given = []
        _approvals_given = d.pop("approvalsGiven")
        for approvals_given_item_data in (_approvals_given):
            approvals_given_item = TeamMemberApproval.from_dict(approvals_given_item_data)



            approvals_given.append(approvals_given_item)


        approvals_requested = []
        _approvals_requested = d.pop("approvalsRequested")
        for approvals_requested_item_data in (_approvals_requested):
            approvals_requested_item = TeamMemberApproval.from_dict(approvals_requested_item_data)



            approvals_requested.append(approvals_requested_item)


        can_approve = d.pop("canApprove")

        can_manage = d.pop("canManage")

        groups = cast(list[str], d.pop("groups"))


        helpers = cast(list[str], d.pop("helpers"))


        member = TeamMember.from_dict(d.pop("member"))




        permissions = []
        _permissions = d.pop("permissions")
        for permissions_item_data in (_permissions):
            permissions_item = TeamEffectivePermission.from_dict(permissions_item_data)



            permissions.append(permissions_item)


        scheduled_campaigns = d.pop("scheduledCampaigns")

        sessions = d.pop("sessions")

        sessions_shared = d.pop("sessionsShared")

        sign_in_methods = []
        _sign_in_methods = d.pop("signInMethods")
        for sign_in_methods_item_data in (_sign_in_methods):
            sign_in_methods_item = MemberSignInMethod.from_dict(sign_in_methods_item_data)



            sign_in_methods.append(sign_in_methods_item)


        keys_note = d.pop("keysNote", UNSET)

        sign_out_reason = d.pop("signOutReason", UNSET)

        sso_domain = d.pop("ssoDomain", UNSET)

        team_member_detail = cls(
            activity=activity,
            api_keys=api_keys,
            approvals_given=approvals_given,
            approvals_requested=approvals_requested,
            can_approve=can_approve,
            can_manage=can_manage,
            groups=groups,
            helpers=helpers,
            member=member,
            permissions=permissions,
            scheduled_campaigns=scheduled_campaigns,
            sessions=sessions,
            sessions_shared=sessions_shared,
            sign_in_methods=sign_in_methods,
            keys_note=keys_note,
            sign_out_reason=sign_out_reason,
            sso_domain=sso_domain,
        )

        return team_member_detail

