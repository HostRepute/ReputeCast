from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.team_member_status import TeamMemberStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.member_sign_in_method import MemberSignInMethod
  from ..models.member_two_fa import MemberTwoFA
  from ..models.scope_summary import ScopeSummary
  from ..models.team_role_ref import TeamRoleRef





T = TypeVar("T", bound="TeamMember")



@_attrs_define
class TeamMember:
    """ 
        Attributes:
            account_deactivated (bool):
            avatar_color (str):
            email (str):
            id (str): Membership id
            joined_at (datetime.datetime):
            last_owner (bool):
            name (str):
            notes (list[str]):
            role (TeamRoleRef):
            scope (ScopeSummary):
            sign_in (list[MemberSignInMethod]):
            source (str):
            status (TeamMemberStatus):
            two_fa (MemberTwoFA):
            user_id (str):
            you (bool):
            deactivated_at (datetime.datetime | Unset):
            last_active_at (datetime.datetime | Unset):
     """

    account_deactivated: bool
    avatar_color: str
    email: str
    id: str
    joined_at: datetime.datetime
    last_owner: bool
    name: str
    notes: list[str]
    role: TeamRoleRef
    scope: ScopeSummary
    sign_in: list[MemberSignInMethod]
    source: str
    status: TeamMemberStatus
    two_fa: MemberTwoFA
    user_id: str
    you: bool
    deactivated_at: datetime.datetime | Unset = UNSET
    last_active_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.member_sign_in_method import MemberSignInMethod # noqa: PLC0415
        from ..models.member_two_fa import MemberTwoFA # noqa: PLC0415
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        from ..models.team_role_ref import TeamRoleRef # noqa: PLC0415
        account_deactivated = self.account_deactivated

        avatar_color = self.avatar_color

        email = self.email

        id = self.id

        joined_at = self.joined_at.isoformat()

        last_owner = self.last_owner

        name = self.name

        notes = self.notes



        role = self.role.to_dict()

        scope = self.scope.to_dict()

        sign_in = []
        for sign_in_item_data in self.sign_in:
            sign_in_item = sign_in_item_data.to_dict()
            sign_in.append(sign_in_item)



        source = self.source

        status = self.status.value

        two_fa = self.two_fa.to_dict()

        user_id = self.user_id

        you = self.you

        deactivated_at: str | Unset = UNSET
        if not isinstance(self.deactivated_at, Unset):
            deactivated_at = self.deactivated_at.isoformat()

        last_active_at: str | Unset = UNSET
        if not isinstance(self.last_active_at, Unset):
            last_active_at = self.last_active_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accountDeactivated": account_deactivated,
            "avatarColor": avatar_color,
            "email": email,
            "id": id,
            "joinedAt": joined_at,
            "lastOwner": last_owner,
            "name": name,
            "notes": notes,
            "role": role,
            "scope": scope,
            "signIn": sign_in,
            "source": source,
            "status": status,
            "twoFA": two_fa,
            "userId": user_id,
            "you": you,
        })
        if deactivated_at is not UNSET:
            field_dict["deactivatedAt"] = deactivated_at
        if last_active_at is not UNSET:
            field_dict["lastActiveAt"] = last_active_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.member_sign_in_method import MemberSignInMethod # noqa: PLC0415
        from ..models.member_two_fa import MemberTwoFA # noqa: PLC0415
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        from ..models.team_role_ref import TeamRoleRef # noqa: PLC0415
        d = dict(src_dict)
        account_deactivated = d.pop("accountDeactivated")

        avatar_color = d.pop("avatarColor")

        email = d.pop("email")

        id = d.pop("id")

        joined_at = datetime.datetime.fromisoformat(d.pop("joinedAt"))




        last_owner = d.pop("lastOwner")

        name = d.pop("name")

        notes = cast(list[str], d.pop("notes"))


        role = TeamRoleRef.from_dict(d.pop("role"))




        scope = ScopeSummary.from_dict(d.pop("scope"))




        sign_in = []
        _sign_in = d.pop("signIn")
        for sign_in_item_data in (_sign_in):
            sign_in_item = MemberSignInMethod.from_dict(sign_in_item_data)



            sign_in.append(sign_in_item)


        source = d.pop("source")

        status = TeamMemberStatus(d.pop("status"))




        two_fa = MemberTwoFA.from_dict(d.pop("twoFA"))




        user_id = d.pop("userId")

        you = d.pop("you")

        _deactivated_at = d.pop("deactivatedAt", UNSET)
        deactivated_at: datetime.datetime | Unset
        if isinstance(_deactivated_at,  Unset):
            deactivated_at = UNSET
        else:
            deactivated_at = datetime.datetime.fromisoformat(_deactivated_at)




        _last_active_at = d.pop("lastActiveAt", UNSET)
        last_active_at: datetime.datetime | Unset
        if isinstance(_last_active_at,  Unset):
            last_active_at = UNSET
        else:
            last_active_at = datetime.datetime.fromisoformat(_last_active_at)




        team_member = cls(
            account_deactivated=account_deactivated,
            avatar_color=avatar_color,
            email=email,
            id=id,
            joined_at=joined_at,
            last_owner=last_owner,
            name=name,
            notes=notes,
            role=role,
            scope=scope,
            sign_in=sign_in,
            source=source,
            status=status,
            two_fa=two_fa,
            user_id=user_id,
            you=you,
            deactivated_at=deactivated_at,
            last_active_at=last_active_at,
        )

        return team_member

